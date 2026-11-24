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
	cmd := command.Parse(line)
	u.notice.SetText("")
	run := func(dir string) { u.runCommand(cmd, dir) }
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
	run(u.workDir)
}

func (u *boardUI) runCommand(cmd command.Command, dir string) error {
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
	return err
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
