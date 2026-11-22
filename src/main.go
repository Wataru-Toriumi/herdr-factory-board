package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func showWorkspaces(herdr string) error {
	workspaces, err := listWorkspaces(herdr)
	if err != nil {
		return err
	}
	fmt.Println("ワークスペース")
	for _, ws := range workspaces {
		marker := " "
		if ws.Focused {
			marker = "*"
		}
		label := ws.Label
		if label == "" {
			label = ws.ID
		}
		fmt.Printf("%s %s  [%s]\n", marker, label, ws.AgentStatus)
	}
	if len(workspaces) == 0 {
		fmt.Println("ワークスペースはありません。")
	}
	return nil
}

func showBoard(s store, herdr string) error {
	b, err := s.read()
	if err != nil {
		return err
	}
	fmt.Println("Factory Board")
	if b.Dir == "" {
		fmt.Println("作業先を設定: dir /path/to/repository")
	} else {
		fmt.Println("作業先:", b.Dir)
	}
	for _, status := range []string{todo, starting, running, done, failed} {
		fmt.Printf("\n%s\n", status)
		for _, c := range b.Cards {
			if c.Status == status {
				fmt.Printf("  #%d  %s  %s\n", c.ID, c.Prompt, c.Source)
			}
		}
	}
	active, err := schedulerRunning(s)
	if err != nil {
		return err
	}
	state := "停止中"
	if active {
		state = "稼働中"
	}
	fmt.Printf("\n毎日の定期実行 (%s)\n", state)
	for _, r := range b.Schedules {
		fmt.Printf("  #%d  %s  %s\n", r.ID, r.Time, r.Prompt)
	}
	fmt.Println()
	return showWorkspaces(herdr)
}

func command(s store, herdr, line string) (string, error) {
	name, arg, _ := strings.Cut(strings.TrimSpace(line), " ")
	arg = strings.TrimSpace(arg)
	switch name {
	case "":
		return "", nil
	case "dir":
		if !filepath.IsAbs(arg) {
			return "", errors.New("作業先の絶対パスを指定してください")
		}
		info, err := os.Stat(arg)
		if err != nil {
			return "", err
		}
		if !info.IsDir() {
			return "", errors.New("作業先はディレクトリを指定してください")
		}
		return "", s.edit(func(b *board) error { b.Dir = filepath.Clean(arg); return nil })
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
	case "daily":
		return "", registerDaily(s, herdr, arg)
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

func runBoard(s store, herdr string) {
	input := bufio.NewScanner(os.Stdin)
	var message string
	for {
		fmt.Print("\x1b[2J\x1b[H")
		if err := showBoard(s, herdr); err != nil {
			fmt.Println(err)
		}
		fmt.Println("\nadd 作業内容 | start 番号 | open 番号 | result 番号")
		fmt.Println("daily HH:MM 作業内容 | remove 定期番号 | scheduler | recover 番号")
		fmt.Println("dir 絶対パス | Enter で更新 | q で終了")
		if message != "" {
			fmt.Println(message)
		}
		fmt.Print("> ")
		if !input.Scan() || strings.EqualFold(strings.TrimSpace(input.Text()), "q") {
			return
		}
		var err error
		message, err = command(s, herdr, input.Text())
		if err != nil {
			message = err.Error()
		}
	}
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
		runBoard(s, herdr)
		return nil
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
