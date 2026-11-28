// Package command はボードの入力コマンドを解析して実行する。
package command

import (
	"errors"
	"strconv"
	"strings"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/application/scheduler"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/application/worker"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/schedule"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/store"
)

type Command struct {
	Name, Arg string
	Agent     string
}

func Parse(line string) Command {
	name, arg, _ := strings.Cut(strings.TrimSpace(line), " ")
	return Command{Name: name, Arg: strings.TrimSpace(arg)}
}

func (cmd Command) Run(s store.Store, herdr, dir string) (*board.Card, error) {
	arg := cmd.Arg
	switch cmd.Name {
	case "":
		return nil, nil
	case "dir":
		return nil, s.RegisterProject(arg)
	case "add":
		if arg == "" {
			return nil, errors.New("作業内容を入力してください")
		}
		return nil, s.Edit(func(b *board.State) error {
			if dir == "" {
				return errors.New("先に dir で作業先を指定してください")
			}
			b.NewCard(arg, dir, "").Agent = b.AgentForNewItem(cmd.Agent)
			return nil
		})
	case schedule.Daily, schedule.Every:
		return nil, scheduler.Register(s, herdr, dir, cmd.Name, arg, cmd.Agent)
	case "scheduler":
		return nil, scheduler.EnsureRunning(s, herdr)
	case "start", "open", "result", "recover", "remove":
		id, err := strconv.Atoi(arg)
		if err != nil || id < 1 {
			return nil, errors.New("番号を指定してください")
		}
		switch cmd.Name {
		case "start":
			return nil, worker.Start(s, herdr, id)
		case "open":
			return nil, worker.Focus(s, herdr, id)
		case "recover":
			return nil, worker.Recover(s, id)
		case "result":
			b, err := s.Read()
			if err != nil {
				return nil, err
			}
			return b.FindCard(id)
		case "remove":
			return nil, scheduler.Remove(s, id)
		}
	}
	return nil, errors.New("コマンドを確認してください")
}
