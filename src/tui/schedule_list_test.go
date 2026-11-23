package tui

import (
	"testing"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/schedule"
)

func TestScheduleListRefreshesLookupOutsideVisibleProject(t *testing.T) {
	list := &scheduleList{List: boardList()}
	api := schedule.Rule{ID: 1, Dir: "/api", Time: "09:00"}
	web := schedule.Rule{ID: 2, Dir: "/web", Time: "10:00"}
	list.render([]schedule.Rule{api, web}, web.Dir, false, nil)

	api.Time = "11:00"
	list.render([]schedule.Rule{api, web}, web.Dir, false, nil)
	if r, ok := list.byID(api.ID); !ok || r.Time != api.Time {
		t.Fatalf("表示対象外の予定の更新が反映されていません: %+v", r)
	}
	if r, ok := list.selected(); !ok || r.ID != web.ID || list.GetItemCount() != 1 {
		t.Fatalf("絞り込み後の一覧と選択: %d件, %+v", list.GetItemCount(), r)
	}
}
