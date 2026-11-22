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
	var attempt int
	err := s.edit(func(b *board) error {
		c, err := b.find(id)
		if err != nil {
			return err
		}
		if c.Status != todo && c.Status != failed {
			return fmt.Errorf("#%d は %s です", id, c.Status)
		}
		c.Attempt++
		attempt = c.Attempt
		c.Status, c.Pane, c.Result = starting, "", ""
		return nil
	})
	if err != nil {
		return err
	}
	err = openPlugin(herdr, "worker", fmt.Sprintf("FACTORY_CARD=%d", id), fmt.Sprintf("FACTORY_ATTEMPT=%d", attempt))
	if err == nil {
		return nil
	}
	launchErr := err
	saveErr := s.edit(func(b *board) error {
		c, err := b.find(id)
		if err == nil && c.Attempt == attempt && c.Status == starting {
			c.Status, c.Result = failed, launchErr.Error()
		}
		return err
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
	err = s.edit(func(b *board) error {
		c, err := b.find(id)
		if err != nil {
			return err
		}
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
	cmd := exec.Command("codex", "exec", "--output-last-message", resultPath, "-")
	cmd.Dir = task.Dir
	cmd.Stdin = strings.NewReader(task.Prompt)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	runErr := cmd.Run()
	msg, readErr := os.ReadFile(resultPath)
	result := strings.TrimSpace(string(msg))
	status := done
	if runErr != nil {
		status = failed
		result = strings.TrimSpace(result + "\n" + runErr.Error())
	} else if readErr != nil || result == "" {
		result = "終了コード 0（最終メッセージなし）"
	}
	return s.edit(func(b *board) error {
		c, err := b.find(id)
		if err == nil {
			c.Status, c.Result = status, result
		}
		return err
	})
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
	return s.edit(func(b *board) error {
		c, err := b.find(id)
		if err != nil {
			return err
		}
		if c.Status != starting && c.Status != running {
			return errors.New("中断した実行だけを復旧できます")
		}
		c.Status, c.Result = failed, "中断した実行を手動で復旧しました"
		return nil
	})
}
