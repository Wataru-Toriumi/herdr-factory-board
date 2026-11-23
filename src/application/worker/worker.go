// Package worker はカードの実行、キャンセル、復旧を担当する。
package worker

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
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
		attempt = c.QueueAttempt()
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
		c.FailAttemptLaunch(attempt, launchErr.Error())
		return nil
	})
	return errors.Join(launchErr, saveErr)
}

func workerLockName(id int) string {
	return fmt.Sprintf("worker-%d.lock", id)
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
	lock, err := s.Lock(workerLockName(id), false)
	if err != nil {
		return err
	}
	defer lock.Close()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGHUP)
	defer stop()
	var task board.Card
	err = s.EditCard(id, func(c *board.Card) error {
		if !c.MarkAttemptRunning(attempt, os.Getenv("HERDR_PANE_ID")) {
			return errors.New("この起動要求は無効です")
		}
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
	status, result := runAgent(ctx, task, resultPath)
	return s.EditCard(id, func(c *board.Card) error {
		c.FinishAttempt(task.Attempt, task.Phase, status, result)
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
			if err == nil && !c.IsRunningAttempt(task.Attempt) {
				cancel()
				return
			}
		}
	}
}

// ワーカーは実行中、カードごとのロックを保持する。
// 終了状態を記録する前に異常終了してもロックは解放されるため、カードを復旧できる。
func Recover(s store.Store, id int) error {
	l, err := s.Lock(workerLockName(id), true)
	if errors.Is(err, syscall.EWOULDBLOCK) {
		return errors.New("担当ペインは実行中です")
	}
	if err != nil {
		return err
	}
	defer l.Close()
	return s.EditCard(id, (*board.Card).RecoverInterruptedAttempt)
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
	pane := c.RunningPane()
	if pane == "" {
		return errors.New("実行中の担当ペインがありません。結果は result で確認できます")
	}
	return herdr.FocusPane(bin, pane)
}
