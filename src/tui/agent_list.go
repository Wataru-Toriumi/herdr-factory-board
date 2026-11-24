package tui

import (
	"github.com/Wataru-Toriumi/herdr-factory-board/src/application/worker"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/herdr"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/tui/agentlist"
	"github.com/gdamore/tcell/v2"
)

type agentList struct {
	*rowList[agentlist.Row]
	blocked int
}

func (l *agentList) render(model agentlist.ViewModel) {
	l.blocked = model.Blocked
	l.setRows(model.Rows, func(a, b agentlist.Row) bool { return a.Pane == b.Pane }, agentlist.Row.Labels)
	l.SetTitle(model.Title())
}

func (u *boardUI) focusAgent(row agentlist.Row) error {
	if row.CardID != 0 {
		return worker.Focus(u.store, u.herdr, row.CardID)
	}
	return herdr.FocusAgent(u.herdr, row.Pane)
}

func (u *boardUI) agentListKey(event *tcell.EventKey) bool {
	if event.Key() != tcell.KeyEnter {
		return false
	}
	if row, ok := u.agents.selected(); ok {
		if err := u.focusAgent(row); err != nil {
			u.notice.SetText(err.Error())
		}
	}
	return true
}
