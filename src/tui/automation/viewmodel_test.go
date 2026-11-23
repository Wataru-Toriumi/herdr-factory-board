package automation

import (
	"errors"
	"testing"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/schedule"
)

func TestFilteredTitleReflectsSchedulerStatus(t *testing.T) {
	rules := []schedule.Rule{{ID: 1, Dir: "/api"}, {ID: 2, Dir: "/web"}}
	for _, tc := range []struct {
		active bool
		err    error
		title  string
	}{
		{false, nil, " 定期実行 1件 · 停止中 "},
		{true, nil, " 定期実行 1件 · 稼働中 "},
		{true, errors.New("状態取得失敗"), " 定期実行 1件 · 状態不明 "},
	} {
		model := Build(rules, "/api", tc.active, tc.err)
		if model.Title != tc.title || len(model.Rules) != 1 || model.Rules[0].ID != 1 {
			t.Errorf("表示 = %q, %+v", model.Title, model.Rules)
		}
	}
}
