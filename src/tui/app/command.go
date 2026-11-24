package app

import (
	"path/filepath"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/application/command"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/application/worker"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/schedule"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/herdr"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/tui/agentlist"
)

func (u *boardUI) execute(line string) {
	u.executeAt(command.Parse(line), "")
}

func (u *boardUI) executeAt(cmd command.Command, dir string) {
	u.notice.SetText("")
	if dir == "" {
		run := func(selected string) { u.runCommand(cmd, selected) }
		switch cmd.Name {
		case "add":
			if cmd.Arg != "" {
				u.pickProject(u.workDir, false, run)
				return
			}
		case schedule.Daily, schedule.Every:
			u.selectScheduleProject(run)
			return
		}
		dir = u.workDir
	}
	u.runCommand(cmd, dir)
}

func (u *boardUI) runCommand(cmd command.Command, dir string) {
	c, err := cmd.Run(u.store, u.herdr, dir)
	if err == nil && cmd.Name == "dir" {
		u.workDir = filepath.Clean(cmd.Arg)
	}
	u.refresh()
	if err != nil {
		u.notice.SetText(err.Error())
	} else if cmd.Name == "add" && u.workDir != "" && dir != u.workDir {
		u.notice.SetText("チケットを追加しました。作業先: " + dir)
	} else if c != nil {
		u.openCardDetail(*c)
	}
}

func (u *boardUI) focusAgent(row agentlist.Row) {
	var err error
	if row.CardID != 0 {
		err = worker.Focus(u.store, u.herdr, row.CardID)
	} else {
		err = herdr.FocusAgent(u.herdr, row.Pane)
	}
	if err != nil {
		u.notice.SetText(err.Error())
	}
}
