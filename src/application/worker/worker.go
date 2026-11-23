// Package worker はカードの実行、キャンセル、復旧を担当する。
package worker

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/herdr"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/store"
)

func Start(s store.Store, herdrBin string, id int) error {
	return launchCard(s, herdrBin, id, func(c *board.Card) error {
		if c.Status != board.Todo && c.Status != board.Failed {
			return fmt.Errorf("#%d は %s です", id, c.Status)
		}
		return nil
	})
}

func launchCard(s store.Store, herdrBin string, id int, prepare func(*board.Card) error) error {
	var attempt int
	err := s.EditCard(id, func(c *board.Card) error {
		if err := prepare(c); err != nil {
			return err
		}
		c.Attempt++
		attempt = c.Attempt
		c.Status, c.Pane, c.Result = board.Starting, "", ""
		return nil
	})
	if err != nil {
		return err
	}
	launchErr := herdr.OpenPlugin(herdrBin, "worker", fmt.Sprintf("FACTORY_CARD=%d", id), fmt.Sprintf("FACTORY_ATTEMPT=%d", attempt))
	if launchErr == nil {
		return nil
	}
	saveErr := s.EditCard(id, func(c *board.Card) error {
		if c.Attempt == attempt && c.Status == board.Starting {
			c.Status, c.Result = board.Failed, launchErr.Error()
		}
		return nil
	})
	return errors.Join(launchErr, saveErr)
}

func Cancel(s store.Store, seen board.Card) error {
	return s.EditCard(seen.ID, func(c *board.Card) error {
		if c.Attempt != seen.Attempt || c.Status != seen.Status || !c.CanCancel() {
			return errors.New("カードが更新されました。最新の状態を確認してください")
		}
		c.Status = board.Canceled
		return nil
	})
}

func Run(s store.Store) error {
	id, err := strconv.Atoi(os.Getenv("FACTORY_CARD"))
	if err != nil {
		return errors.New("ボードから実行してください")
	}
	attempt, err := strconv.Atoi(os.Getenv("FACTORY_ATTEMPT"))
	if err != nil {
		return err
	}
	lock, err := s.Lock(fmt.Sprintf("worker-%d.lock", id), false)
	if err != nil {
		return err
	}
	defer lock.Close()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGHUP)
	defer stop()
	var task board.Card
	err = s.EditCard(id, func(c *board.Card) error {
		if c.Attempt != attempt || c.Status != board.Starting {
			return errors.New("この起動要求は無効です")
		}
		c.Status, c.Pane = board.Running, os.Getenv("HERDR_PANE_ID")
		task = *c
		return nil
	})
	if err != nil {
		return err
	}
	fmt.Printf("#%d  %s\n作業先: %s\n\n", id, task.Prompt, task.Dir)
	resultPath := filepath.Join(s.Dir, fmt.Sprintf(".result-%d-%d", id, attempt))
	defer os.Remove(resultPath)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go watchCardCancellation(ctx, cancel, s, task)
	status, result := runCodex(ctx, task, resultPath)
	return s.EditCard(id, func(c *board.Card) error {
		if c.Attempt != attempt || c.Status != board.Running {
			return nil
		}
		c.Status, c.Result = status, result
		if status == board.Done {
			switch task.Phase {
			case board.DesignPhase:
				c.Design, c.Status, c.Result = result, board.AwaitingDesignReview, ""
			case board.ImplementationPhase:
				c.Status = board.AwaitingResultReview
			}
		}
		return nil
	})
}

// ボードとワーカーは別のプロセスなので、保存されたカードを見て中止要求を受け取る。
func watchCardCancellation(ctx context.Context, cancel context.CancelFunc, s store.Store, task board.Card) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			b, err := s.Read()
			if err != nil {
				continue
			}
			c, err := b.FindCard(task.ID)
			if err == nil && (c.Attempt != task.Attempt || c.Status != board.Running) {
				cancel()
				return
			}
		}
	}
}

func runCodex(ctx context.Context, task board.Card, resultPath string) (status, result string) {
	args := []string{"exec", "--output-last-message", resultPath}
	if task.Phase == board.DesignPhase {
		args = append(args, "--sandbox", "read-only")
	}
	cmd := exec.CommandContext(ctx, "codex", append(args, "-")...)
	// Codex が起動したコマンドも一緒に止められるよう、専用のプロセスグループで実行する。
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
	cmd.Dir = task.Dir
	cmd.Stdin = strings.NewReader(workerPrompt(task))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	runErr := cmd.Run()
	msg, readErr := os.ReadFile(resultPath)
	result = strings.TrimSpace(string(msg))
	if ctx.Err() != nil {
		return board.Canceled, result
	}
	var exitErr *exec.ExitError
	if errors.As(runErr, &exitErr) && (exitErr.ExitCode() == 130 || exitErr.Sys().(syscall.WaitStatus).Signal() == syscall.SIGINT) {
		return board.Canceled, result
	}
	if runErr != nil {
		return board.Failed, strings.TrimSpace(result + "\n" + runErr.Error())
	}
	if readErr != nil || result == "" {
		if task.Phase == board.DesignPhase {
			return board.Failed, "設計案を取得できませんでした"
		}
		return board.Done, "終了コード 0（最終メッセージなし）"
	}
	return board.Done, result
}

// ワーカーは実行中、カードごとのロックを保持する。
// 終了状態を記録する前に異常終了してもロックは解放されるため、カードを復旧できる。
func Recover(s store.Store, id int) error {
	l, err := s.Lock(fmt.Sprintf("worker-%d.lock", id), true)
	if errors.Is(err, syscall.EWOULDBLOCK) {
		return errors.New("担当ペインは実行中です")
	}
	if err != nil {
		return err
	}
	defer l.Close()
	return s.EditCard(id, func(c *board.Card) error {
		if c.Status != board.Starting && c.Status != board.Running {
			return errors.New("中断した実行だけを復旧できます")
		}
		c.Status, c.Result = board.Failed, "中断した実行を手動で復旧しました"
		return nil
	})
}

func Focus(s store.Store, bin string, id int) error {
	b, err := s.Read()
	if err != nil {
		return err
	}
	c, err := b.FindCard(id)
	if err != nil {
		return err
	}
	if c.Status != board.Running || c.Pane == "" {
		return errors.New("実行中の担当ペインがありません。結果は result で確認できます")
	}
	return herdr.FocusPane(bin, c.Pane)
}
