package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

func startCard(s store, herdr string, id int) error {
	return launchCard(s, herdr, id, func(c *card) error {
		if c.Status != todo && c.Status != failed {
			return fmt.Errorf("#%d は %s です", id, c.Status)
		}
		return nil
	})
}

func launchCard(s store, herdr string, id int, prepare func(*card) error) error {
	var attempt int
	err := s.editCard(id, func(c *card) error {
		if err := prepare(c); err != nil {
			return err
		}
		c.Attempt++
		attempt = c.Attempt
		c.Status, c.Pane, c.Result = starting, "", ""
		return nil
	})
	if err != nil {
		return err
	}
	launchErr := openPlugin(herdr, "worker", fmt.Sprintf("FACTORY_CARD=%d", id), fmt.Sprintf("FACTORY_ATTEMPT=%d", attempt))
	if launchErr == nil {
		return nil
	}
	saveErr := s.editCard(id, func(c *card) error {
		if c.Attempt == attempt && c.Status == starting {
			c.Status, c.Result = failed, launchErr.Error()
		}
		return nil
	})
	return errors.Join(launchErr, saveErr)
}

func runWorker(s store) error {
	id, err := strconv.Atoi(os.Getenv("FACTORY_CARD"))
	if err != nil {
		return errors.New("ボードから実行してください")
	}
	attempt, err := strconv.Atoi(os.Getenv("FACTORY_ATTEMPT"))
	if err != nil {
		return err
	}
	lock, err := s.lock(fmt.Sprintf("worker-%d.lock", id), false)
	if err != nil {
		return err
	}
	defer lock.Close()
	var task card
	err = s.editCard(id, func(c *card) error {
		if c.Attempt != attempt || c.Status != starting {
			return errors.New("この起動要求は無効です")
		}
		c.Status, c.Pane = running, os.Getenv("HERDR_PANE_ID")
		task = *c
		return nil
	})
	if err != nil {
		return err
	}
	fmt.Printf("#%d  %s\n作業先: %s\n\n", id, task.Prompt, task.Dir)
	resultPath := filepath.Join(s.dir, fmt.Sprintf(".result-%d-%d", id, attempt))
	defer os.Remove(resultPath)
	status, result := runCodex(task, resultPath)
	return s.editCard(id, func(c *card) error {
		if c.Attempt != attempt || c.Status != running {
			return nil
		}
		c.Status, c.Result = status, result
		if status == done {
			switch task.Phase {
			case designPhase:
				c.Design, c.Status, c.Result = result, awaitingReview, ""
			case implementationPhase:
				c.Status = awaitingResultReview
			}
		}
		return nil
	})
}

func runCodex(task card, resultPath string) (status, result string) {
	args := []string{"exec", "--output-last-message", resultPath}
	if task.Phase == designPhase {
		args = append(args, "--sandbox", "read-only")
	}
	cmd := exec.Command( "codex", append(args, "-")...)
	cmd.Dir = task.Dir
	cmd.Stdin = strings.NewReader(task.workerPrompt())
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	runErr := cmd.Run()
	msg, readErr := os.ReadFile(resultPath)
	result = strings.TrimSpace(string(msg))
	if runErr != nil {
		return failed, strings.TrimSpace(result + "\n" + runErr.Error())
	}
	if readErr != nil || result == "" {
		if task.Phase == designPhase {
			return failed, "設計案を取得できませんでした"
		}
		return done, "終了コード 0（最終メッセージなし）"
	}
	return done, result
}

// ワーカーは実行中、カードごとのロックを保持する。
// 終了状態を記録する前に異常終了してもロックは解放されるため、カードを復旧できる。
func recoverCard(s store, id int) error {
	l, err := s.lock(fmt.Sprintf("worker-%d.lock", id), true)
	if errors.Is(err, syscall.EWOULDBLOCK) {
		return errors.New("担当ペインは実行中です")
	}
	if err != nil {
		return err
	}
	defer l.Close()
	return s.editCard(id, func(c *card) error {
		if c.Status != starting && c.Status != running {
			return errors.New("中断した実行だけを復旧できます")
		}
		c.Status, c.Result = failed, "中断した実行を手動で復旧しました"
		return nil
	})
}
