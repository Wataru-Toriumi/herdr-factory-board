package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type boardCommand struct {
	name, arg string
}

func parseCommand(line string) boardCommand {
	name, arg, _ := strings.Cut(strings.TrimSpace(line), " ")
	return boardCommand{name: name, arg: strings.TrimSpace(arg)}
}

func (cmd boardCommand) run(s store, herdr, dir string) (*card, error) {
	arg := cmd.arg
	switch cmd.name {
	case "":
		return nil, nil
	case "dir":
		return nil, registerProject(s, arg)
	case "add":
		if arg == "" {
			return nil, errors.New("作業内容を入力してください")
		}
		return nil, s.edit(func(b *board) error {
			if dir == "" {
				return errors.New("先に dir で作業先を指定してください")
			}
			b.newCard(arg, dir, "")
			return nil
		})
	case "daily", "every":
		return nil, registerSchedule(s, herdr, dir, cmd.name, arg)
	case "scheduler":
		return nil, ensureScheduler(s, herdr)
	case "start", "open", "result", "recover", "remove":
		id, err := strconv.Atoi(arg)
		if err != nil || id < 1 {
			return nil, errors.New("番号を指定してください")
		}
		switch cmd.name {
		case "start":
			return nil, startCard(s, herdr, id)
		case "open":
			return nil, focusWorker(s, herdr, id)
		case "recover":
			return nil, recoverCard(s, id)
		case "result":
			b, err := s.read()
			if err != nil {
				return nil, err
			}
			return b.find(id)
		case "remove":
			return nil, s.edit(func(b *board) error {
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
	return nil, errors.New("コマンドを確認してください")
}
