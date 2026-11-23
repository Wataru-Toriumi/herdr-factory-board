package tui

import (
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/schedule"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/tui/automation"
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
