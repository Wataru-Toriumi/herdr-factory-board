// Package automation は定期実行の一覧、設定フォーム、画面内のキー操作を担当する。
package automation

import (
	"fmt"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/schedule"
)

type ViewModel struct {
	Rules    []schedule.Rule
	Title    string
	allRules []schedule.Rule
}

// ByID はプロジェクトの絞り込みに関わらず、カードの元の予定を検索する。
func (m ViewModel) ByID(id int) (schedule.Rule, bool) {
	i := schedule.IndexByID(m.allRules, id)
	if i < 0 {
		return schedule.Rule{}, false
	}
	return m.allRules[i], true
}

func Build(allRules []schedule.Rule, dir string, active bool, err error) ViewModel {
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
	return ViewModel{Rules: visibleRules, Title: fmt.Sprintf(" 定期実行 %d件 · %s ", len(visibleRules), state), allRules: allRules}
}

func RowLabels(r schedule.Rule) (string, string) {
	return fmt.Sprintf("%s  %s", r.Frequency(), r.Prompt), fmt.Sprintf("#%d  %s", r.ID, r.Dir)
}
