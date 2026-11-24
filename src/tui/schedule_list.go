package tui

import (
	"fmt"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/schedule"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/tui/automation"
	"github.com/gdamore/tcell/v2"
)

type scheduleList struct {
	*rowList[schedule.Rule]
	model automation.ViewModel
}

func (l *scheduleList) render(rules []schedule.Rule, dir string, active bool, err error) {
	l.model = automation.Build(rules, dir, active, err)
	l.SetTitle(l.model.Title)
	l.setRows(l.model.Rules, func(a, b schedule.Rule) bool { return a.ID == b.ID }, automation.RowLabels)
}

func (u *boardUI) scheduleListKey(event *tcell.EventKey) bool {
	switch event.Key() {
	case tcell.KeyEnter:
		if r, ok := u.schedules.selected(); ok {
			u.openDetail(fmt.Sprintf("定期実行 #%d", r.ID), "", r.ID, nil)
		}
	case tcell.KeyRune:
		switch event.Rune() {
		case 'n':
			u.newScheduleForm(schedule.Daily)
		case 'i':
			u.newScheduleForm(schedule.Every)
		case 't':
			if r, ok := u.schedules.selected(); ok {
				u.editScheduleTiming(r)
			}
		case 'x':
			if r, ok := u.schedules.selected(); ok {
				u.editCommand(fmt.Sprintf("remove %d", r.ID))
			}
		default:
			return false
		}
	default:
		return false
	}
	return true
}
