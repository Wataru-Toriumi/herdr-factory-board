package tui

import (
	"fmt"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/schedule"
)

type scheduleList struct {
	*rowList[schedule.Rule]
	allRules []schedule.Rule
}

func (l *scheduleList) byID(id int) (schedule.Rule, bool) {
	i := schedule.IndexByID(l.allRules, id)
	if i < 0 {
		return schedule.Rule{}, false
	}
	return l.allRules[i], true
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
	l.setRows(visibleRules, func(a, b schedule.Rule) bool { return a.ID == b.ID }, scheduleRowLabels)
}

func scheduleRowLabels(r schedule.Rule) (string, string) {
	return fmt.Sprintf("%s  %s", r.Frequency(), r.Prompt), fmt.Sprintf("#%d  %s", r.ID, r.Dir)
}
