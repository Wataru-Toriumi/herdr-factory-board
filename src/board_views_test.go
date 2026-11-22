package main

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestCardSelectionFollowsWorkerAcrossColumns(t *testing.T) {
	u := newBoardUI(store{t.TempDir()}, "")
	cards := []card{{ID: 1, Status: todo}, {ID: 2, Status: todo}, {ID: 3, Status: done}}
	u.renderTasks(cards)
	u.lists[0].SetCurrentItem(1)
	for _, status := range []string{starting, running, done} {
		cards[1].Status = status
		u.renderTasks(cards)
		selected, ok := u.selectedCard()
		if !ok || selected.ID != 2 || selected.Status != status {
			t.Fatalf("更新後の選択: %+v, %v", selected, ok)
		}
	}
}

func TestRefreshPreservesActiveView(t *testing.T) {
	for _, tc := range []struct {
		name string
		key  rune
		page string
	}{
		{"コマンド入力", ':', "tasks"},
		{"詳細", '?', "detail"},
		{"作業先の選択", 'p', "directory"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := store{t.TempDir()}
			u := newBoardUI(s, "")
			u.board.Dir = s.dir
			cards := []card{{ID: 1, Status: todo}}
			u.renderTasks(cards)
			u.key(tcell.NewEventKey(tcell.KeyRune, tc.key, 0))
			page, view := u.pages.GetFrontPage()
			if page != tc.page {
				t.Fatalf("表示中のページ = %q, 期待値 = %q", page, tc.page)
			}
			if tc.key == ':' {
				view = u.input
			}
			cards[0].Status = running
			u.renderTasks(cards)
			if u.app.GetFocus() != view {
				t.Fatal("カードの列移動で操作中の画面からフォーカスが外れました")
			}
			u.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
			if u.app.GetFocus() != u.lists[cardLane(running)] {
				t.Fatal("Esc で進行中のカードに戻れませんでした")
			}
		})
	}
}
