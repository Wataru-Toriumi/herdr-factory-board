package tui

import (
	"testing"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/schedule"
)

func TestScheduleListRefreshesLookupOutsideVisibleProject(t *testing.T) {
	list := &scheduleList{rowList: newRowList[schedule.Rule]()}
	api := schedule.Rule{ID: 1, Dir: "/api", Time: "09:00"}
	web := schedule.Rule{ID: 2, Dir: "/web", Time: "10:00"}
	list.render([]schedule.Rule{api, web}, web.Dir, false, nil)

	api.Time = "11:00"
	list.render([]schedule.Rule{api, web}, web.Dir, false, nil)
	if r, ok := list.model.ByID(api.ID); !ok || r.Time != api.Time {
		t.Fatalf("表示対象外の予定の更新が反映されていません: %+v", r)
	}
	if r, ok := list.selected(); !ok || r.ID != web.ID || list.GetItemCount() != 1 {
		t.Fatalf("絞り込み後の一覧と選択: %d件, %+v", list.GetItemCount(), r)
	}
}

func TestScheduleSelectionAfterSelectedRuleDisappears(t *testing.T) {
	list := &scheduleList{rowList: newRowList[schedule.Rule]()}
	rules := []schedule.Rule{{ID: 1}, {ID: 2}, {ID: 3}, {ID: 4}}
	list.render(rules, "", false, nil)
	list.SetCurrentItem(2)

	list.render([]schedule.Rule{rules[0], rules[1], rules[3]}, "", false, nil)
	if r, ok := list.selected(); !ok || r.ID != 4 {
		t.Fatalf("選択した予定が消えた後の同じ行: %+v", r)
	}
	list.render(rules[:2], "", false, nil)
	if r, ok := list.selected(); !ok || r.ID != 2 {
		t.Fatalf("末尾の予定が消えた後の選択: %+v", r)
	}
	list.render(nil, "", false, nil)
	if r, ok := list.selected(); ok {
		t.Fatalf("空の一覧に選択が残っています: %+v", r)
	}
}
