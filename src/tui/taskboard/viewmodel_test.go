package taskboard

import (
	"testing"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
)

func TestBuildFollowsSelectedCardAcrossLanes(t *testing.T) {
	cards := []board.Card{
		{ID: 1, Dir: "/web", Status: board.AwaitingResultReview},
		{ID: 2, Dir: "/api", Status: board.AwaitingResultReview},
		{ID: 3, Dir: "/api", Status: board.AwaitingResultReview},
	}
	model := Manual.Build(cards, nil, "/api", Work, 3)
	if rows := model.Rows[Review]; len(rows) != 2 || rows[0].Card.ID != 2 || rows[1].Card.ID != 3 {
		t.Fatalf("絞り込み後のレビュー中の行 = %+v", rows)
	}
	if model.SelectedLane != Review || model.SelectedRow != 1 {
		t.Fatalf("移動後の選択位置 = (%d, %d)", model.SelectedLane, model.SelectedRow)
	}

	model = Manual.Build(cards, nil, "/web", Work, 3)
	if model.SelectedLane != Work || model.SelectedRow != -1 {
		t.Fatalf("選択中のカードが表示対象外になった後の選択位置 = (%d, %d)", model.SelectedLane, model.SelectedRow)
	}
}

func TestRowLabelsDistinguishProjectsWithTheSameName(t *testing.T) {
	dirs := []string{"/backend/api", "/frontend/api"}
	cards := []board.Card{
		{ID: 1, Dir: dirs[0], Status: board.Todo},
		{ID: 2, Dir: dirs[1], Status: board.Todo},
	}
	model := Manual.Build(cards, dirs, "", Todo, 1)
	for i, want := range []string{"#1  /backend/api", "#2  /frontend/api"} {
		if _, got := model.Rows[Todo][i].Labels(); got != want {
			t.Errorf("行 %d のラベル = %q, 期待値 = %q", i, got, want)
		}
	}
}
