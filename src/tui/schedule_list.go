package tui

import (
	"fmt"
	"reflect"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/schedule"
	"github.com/rivo/tview"
)

type scheduleList struct {
	*tview.List
	allRules     []schedule.Rule
	visibleRules []schedule.Rule
}

func (l *scheduleList) byID(id int) (schedule.Rule, bool) {
	i := schedule.IndexByID(l.allRules, id)
	if i < 0 {
		return schedule.Rule{}, false
	}
	return l.allRules[i], true
}

func (l *scheduleList) selected() (schedule.Rule, bool) {
	i := l.GetCurrentItem()
	if i < 0 || i >= len(l.visibleRules) {
		return schedule.Rule{}, false
	}
	return l.visibleRules[i], true
}

func (l *scheduleList) render(allRules []schedule.Rule, dir string, active bool, err error) {
	l.allRules = allRules
	visibleRules := allRules
	if dir != "" {
		visibleRules = nil
		for _, r := range allRules {
			if r.Dir == dir {
				visibleRules = append(visibleRules, r)
			}
		}
	}
	state := "停止中"
	if err != nil {
		state = "状態不明"
	} else if active {
		state = "稼働中"
	}
	l.SetTitle(fmt.Sprintf(" 定期実行 %d件 · %s ", len(visibleRules), state))
	if reflect.DeepEqual(l.visibleRules, visibleRules) {
		return
	}

	selected, _ := l.selected()
	index := l.GetCurrentItem()
	l.Clear()
	for i, r := range visibleRules {
		l.AddItem(tview.Escape(fmt.Sprintf("%s  %s", r.Frequency(), r.Prompt)), tview.Escape(fmt.Sprintf("#%d  %s", r.ID, r.Dir)), 0, nil)
		if r.ID == selected.ID {
			index = i
		}
	}
	if len(visibleRules) > 0 {
		l.SetCurrentItem(min(max(0, index), len(visibleRules)-1))
	}
	l.visibleRules = visibleRules
}
