package main

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
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
			u.workDir = s.dir
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

func TestAutomationTabManagesDailySchedule(t *testing.T) {
	s := store{t.TempDir()}
	t.Setenv("HOME", s.dir)
	u := newBoardUI(s, "true")
	press := func(key tcell.Key) { u.key(tcell.NewEventKey(key, 0, 0)) }
	typeKey := func(ch rune) { u.key(tcell.NewEventKey(tcell.KeyRune, ch, 0)) }

	press(tcell.KeyBacktab)
	typeKey('n')
	page, picker := u.pages.GetFrontPage()
	if page != "directory" {
		t.Fatal("初回の登録で作業先を選択できません")
	}
	picker.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), func(p tview.Primitive) { u.app.SetFocus(p) })
	if u.activePage() != "input" || u.input.GetText() != "daily " {
		t.Fatal("作業先の選択後に定期実行を入力できません")
	}
	u.input.SetText("daily 09:00 テスト")
	press(tcell.KeyEnter)
	b, err := s.read()
	if err != nil {
		t.Fatal(err)
	}
	r, ok := u.schedules.selected()
	if !ok || r.Prompt != "テスト" || r.Dir != s.dir || r.Time != "09:00" || len(b.Cards) != 0 {
		t.Fatalf("登録後のボード: %+v", b)
	}

	press(tcell.KeyEnter)
	if u.activePage() != "detail" {
		t.Fatal("予定の詳細を開けません")
	}
	press(tcell.KeyEscape)
	typeKey('x')
	press(tcell.KeyEscape)
	if u.activePage() != "automation" || u.schedules.GetItemCount() != 1 {
		t.Fatal("解除を取り消した後の一覧が不正です")
	}
	typeKey('x')
	press(tcell.KeyEnter)
	b, err = s.read()
	if err != nil {
		t.Fatal(err)
	}
	if u.schedules.GetItemCount() != 0 || len(b.Schedules) != 0 {
		t.Fatal("確定した予定を解除できません")
	}
	for _, page := range []string{"tasks", "agents", "automation"} {
		press(tcell.KeyTab)
		if u.activePage() != page {
			t.Fatalf("タブ切替後のページ = %q, 期待値 = %q", u.activePage(), page)
		}
	}
}

func TestScheduleSelectionAfterAnotherPaneRemovesRule(t *testing.T) {
	s := store{t.TempDir()}
	if err := s.edit(func(b *board) error {
		b.Schedules = []schedule{{ID: 1}, {ID: 2}, {ID: 3}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	u := newBoardUI(s, "true")
	u.refresh()
	u.key(tcell.NewEventKey(tcell.KeyBacktab, 0, 0))
	u.schedules.SetCurrentItem(1)
	if _, err := command(s, "true", "remove 1"); err != nil {
		t.Fatal(err)
	}
	u.refresh()
	if count := u.schedules.GetItemCount(); count != 2 {
		t.Fatalf("更新後の予定数 = %d", count)
	}
	u.key(tcell.NewEventKey(tcell.KeyRune, 'x', 0))
	if got := u.input.GetText(); got != "remove 2" {
		t.Fatalf("更新後の解除コマンド = %q", got)
	}
}
