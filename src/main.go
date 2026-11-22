package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func setWorkDir(s store, path string) error {
	if !filepath.IsAbs(path) {
		return errors.New("作業先の絶対パスを指定してください")
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("作業先はディレクトリを指定してください")
	}
	return s.edit(func(b *board) error { b.Dir = filepath.Clean(path); return nil })
}

func command(s store, herdr, line string) (string, error) {
	name, arg, _ := strings.Cut(strings.TrimSpace(line), " ")
	arg = strings.TrimSpace(arg)
	switch name {
	case "":
		return "", nil
	case "dir":
		return "", setWorkDir(s, arg)
	case "add":
		if arg == "" {
			return "", errors.New("作業内容を入力してください")
		}
		return "", s.edit(func(b *board) error {
			if b.Dir == "" {
				return errors.New("先に dir で作業先を指定してください")
			}
			b.newCard(arg, b.Dir, "")
			return nil
		})
	case "daily", "every":
		return "", registerSchedule(s, herdr, name, arg)
	case "scheduler":
		return "", ensureScheduler(s, herdr)
	case "start", "open", "result", "recover", "remove":
		id, err := strconv.Atoi(arg)
		if err != nil || id < 1 {
			return "", errors.New("番号を指定してください")
		}
		switch name {
		case "start":
			return "", startCard(s, herdr, id)
		case "open":
			return "", focusWorker(s, herdr, id)
		case "recover":
			return "", recoverCard(s, id)
		case "result":
			b, err := s.read()
			if err != nil {
				return "", err
			}
			c, err := b.find(id)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("#%d  %s  %s\n%s\n%s", c.ID, c.Status, c.Dir, c.Prompt, c.Result), nil
		case "remove":
			return "", s.edit(func(b *board) error {
				for i, r := range b.Schedules {
					if r.ID == id {
						b.Schedules = append(b.Schedules[:i], b.Schedules[i+1:]...)
						return nil
					}
				}
				return fmt.Errorf("定期実行 #%d がありません", id)
			})
		}
	}
	return "", errors.New("コマンドを確認してください")
}

func run() error {
	s, err := workspaceStore()
	if err != nil {
		return err
	}
	herdr := os.Getenv("HERDR_BIN_PATH")
	if herdr == "" {
		herdr = "herdr"
	}
	if len(os.Args) == 1 {
		return runBoard(s, herdr)
	}
	switch os.Args[1] {
	case "worker":
		return runWorker(s)
	case "scheduler":
		return runScheduler(s, herdr)
	default:
		return errors.New("不明な起動モードです")
	}
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
