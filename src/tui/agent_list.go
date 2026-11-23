package tui

import (
	"fmt"
	"reflect"

	"github.com/rivo/tview"
)

type agentList struct {
	*tview.List
	rows []agentRow
}

func (l *agentList) selected() (agentRow, bool) {
	i := l.GetCurrentItem()
	if i < 0 || i >= len(l.rows) {
		return agentRow{}, false
	}
	return l.rows[i], true
}

func (l *agentList) blockedCount() int {
	blocked := 0
	for _, r := range l.rows {
		if r.status == "blocked" {
			blocked++
		}
	}
	return blocked
}

var agentStates = map[string]string{
	"blocked": "入力待ち", "working": "作業中", "idle": "待機中", "done": "完了",
}

func (l *agentList) render(rows []agentRow) {
	if !reflect.DeepEqual(rows, l.rows) {
		index := l.GetCurrentItem()
		selected, _ := l.selected()
		l.Clear()
		for i, r := range rows {
			if r.pane == selected.pane {
				index = i
			}
			state := agentStates[r.status]
			if state == "" {
				state = "状態不明"
			}
			l.AddItem(tview.Escape(fmt.Sprintf("%s  ·  %s   %s", state, r.name, r.task)), tview.Escape(r.workspace+"   "+r.pane), 0, nil)
		}
		if len(rows) > 0 {
			l.SetCurrentItem(min(max(index, 0), len(rows)-1))
		}
		l.rows = rows
	}
	l.SetTitle(fmt.Sprintf(" エージェント %d ", len(rows)))
	if len(rows) == 0 {
		l.SetTitle(" エージェントなし ")
	}
}
