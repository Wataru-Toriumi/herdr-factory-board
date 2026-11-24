package taskboard

import (
	"strings"
	"testing"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestCardSelectionFollowsWorkerAcrossColumns(t *testing.T) {
	view := NewView(nil, func(tview.Primitive) {})
	cards := []board.Card{{ID: 1, Status: board.Todo}, {ID: 2, Status: board.Todo}, {ID: 3, Status: board.Done}}
	view.Render(cards, nil, "")
	view.List(0).SetCurrentItem(1)
	for _, status := range []string{board.Starting, board.Running, board.Done} {
		cards[1].Status = status
		view.Render(cards, nil, "")
		selected, ok := view.SelectedCard()
		if !ok || selected.ID != 2 || selected.Status != status {
			t.Fatalf("更新後の選択: %+v, %v", selected, ok)
		}
	}
}

func TestTaskLabelsRefreshWhenOnlyProjectNamesChange(t *testing.T) {
	view := NewView(nil, func(tview.Primitive) {})
	projects := []string{"/backend/api"}
	cards := []board.Card{{ID: 1, Status: board.Todo, Dir: projects[0]}}
	view.Render(cards, projects, "")
	list := view.List(Todo)
	if _, label := list.GetItemText(0); label != "#1  api" {
		t.Fatalf("登録時の表示 = %q", label)
	}

	projects = append(projects, "/frontend/api")
	view.Render(cards, projects, "")
	if _, label := list.GetItemText(0); label != "#1  /backend/api" {
		t.Fatalf("同名プロジェクト追加後の表示 = %q", label)
	}
}

func TestFailedAndCanceledCardsShareLane(t *testing.T) {
	view := NewView(nil, func(tview.Primitive) {})
	view.Render([]board.Card{{ID: 1, Status: board.Failed, Source: "定期 #3"}, {ID: 2, Status: board.Canceled}}, nil, "")
	list := view.List(FailedOrCanceled)
	if list.GetItemCount() != 2 {
		t.Fatalf("失敗・中止のカード数 = %d", list.GetItemCount())
	}
	for i, want := range []string{"失敗", "キャンセル済み"} {
		_, label := list.GetItemText(i)
		if !strings.HasPrefix(label, want) {
			t.Fatalf("カードの状態表示 = %q, 期待値 = %q", label, want)
		}
	}
}

func TestTaskLaneTitlesAndSelectionColors(t *testing.T) {
	view := NewView(nil, func(tview.Primitive) {})
	view.Render(nil, nil, "")
	for _, want := range []struct {
		title string
		color tcell.Color
	}{
		{" 未着手 0 ", tcell.ColorLightSkyBlue},
		{" 設計中 0 ", tcell.ColorPlum},
		{" 設計完了 0 ", tcell.ColorLavender},
		{" 進行中 0 ", tcell.ColorGold},
		{" レビュー中 0 ", tcell.ColorLightCyan},
		{" 完了 0 ", tcell.ColorLightGreen},
		{" 失敗・中止 0 ", tcell.ColorLightCoral},
	} {
		list := view.List(view.Lane())
		if list.GetTitle() != want.title || list.GetBorderColor() != want.color {
			t.Fatalf("選択中の列 = %q (%v), 期待値 = %q (%v)", list.GetTitle(), list.GetBorderColor(), want.title, want.color)
		}
		view.HandleKey(tcell.NewEventKey(tcell.KeyRight, 0, 0))
	}
	if view.Lane() != Todo {
		t.Fatal("最後の列から未着手に戻れませんでした")
	}
}
