package main

import (
	"strings"
	"testing"
	"time"

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

func TestFailedAndCanceledCardsShareLane(t *testing.T) {
	u := newBoardUI(store{t.TempDir()}, "")
	u.renderTasks([]card{{ID: 1, Status: failed, Source: "定期 #3"}, {ID: 2, Status: canceled}})
	list := u.lists[failedOrCanceledLane]
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
			cards := []card{{ID: 1, Status: todo, Dir: s.dir}}
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
			if u.app.GetFocus() != u.lists[workLane] {
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
	if _, err := parseCommand("remove 1").run(s, "true", s.dir); err != nil {
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

func TestAutomationTabIntervalShortcut(t *testing.T) {
	s := store{t.TempDir()}
	if err := registerProject(s, s.dir); err != nil {
		t.Fatal(err)
	}
	u := newBoardUI(s, "true")
	u.refresh()
	u.key(tcell.NewEventKey(tcell.KeyBacktab, 0, 0))
	u.key(tcell.NewEventKey(tcell.KeyRune, 'i', 0))
	if u.input.GetText() != "every " || u.activePage() != "input" {
		t.Fatal("間隔指定の入力を開始できません")
	}
	u.input.SetText(u.input.GetText() + "5m 変更を確認する")
	u.key(tcell.NewEventKey(tcell.KeyEnter, 0, 0))
	if u.schedules.GetItemCount() != 1 {
		t.Fatal("登録した予定が表示されません")
	}
	text, _ := u.schedules.GetItemText(0)
	if text != "5分おき  変更を確認する" {
		t.Fatalf("一覧の予定: %s", text)
	}
}

func TestEditIntervalFromAutomationTab(t *testing.T) {
	s := store{t.TempDir()}
	original := time.Now().Add(time.Minute)
	if err := s.edit(func(b *board) error {
		b.NextID = 3
		b.Cards = []card{{ID: 3, Prompt: "履歴", Status: done}}
		b.Schedules = []schedule{
			{ID: 1, Time: "09:00"},
			{ID: 2, Prompt: "差分の確認", Dir: s.dir, Interval: &scheduleInterval{Minutes: 1, NextRun: original}},
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	u := newBoardUI(s, "true")
	u.refresh()
	press := func(key tcell.Key) { u.key(tcell.NewEventKey(key, 0, 0)) }
	edit := func() { u.key(tcell.NewEventKey(tcell.KeyRune, 't', 0)) }
	press(tcell.KeyBacktab)
	u.schedules.SetCurrentItem(1)
	edit()
	if u.input.GetText() != "1m" {
		t.Fatal("現在の間隔が入力欄にありません")
	}
	u.input.SetText("30m")
	press(tcell.KeyEscape)
	edit()
	if u.input.GetText() != "1m" {
		t.Fatal("キャンセルした変更が保存されました")
	}
	u.input.SetText("0m")
	press(tcell.KeyEnter)
	if u.activePage() != "input" || u.notice.GetText(false) == "" {
		t.Fatal("入力エラーを修正できません")
	}
	// 編集中に先頭の予定が消えても、保存対象は同じ番号の予定。
	if _, err := parseCommand("remove 1").run(s, "true", s.dir); err != nil {
		t.Fatal(err)
	}
	u.refresh()
	before := time.Now()
	u.input.SetText("15m")
	press(tcell.KeyEnter)
	b, err := s.read()
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Schedules) != 1 || len(b.Cards) != 1 || b.Cards[0].ID != 3 || b.NextID != 3 {
		t.Fatalf("更新後の履歴: %+v", b)
	}
	r := b.Schedules[0]
	if u.activePage() != "automation" || r.ID != 2 || r.Prompt != "差分の確認" || r.Dir != s.dir || r.Interval.Minutes != 15 {
		t.Fatalf("更新後の予定: %+v", r)
	}
	if r.Interval.NextRun.Before(before.Add(15*time.Minute)) || r.Interval.NextRun.After(time.Now().Add(15*time.Minute)) || due(r, original) {
		t.Fatalf("次回の実行時刻: %v", r.Interval.NextRun)
	}
	edit()
	press(tcell.KeyEnter)
	same, ok := u.schedules.selected()
	if !ok || !same.Interval.NextRun.Equal(r.Interval.NextRun) {
		t.Fatal("同じ間隔で確定すると実行時刻がずれました")
	}
}

func TestTimingFromTaskDetail(t *testing.T) {
	s := store{t.TempDir()}
	if err := s.edit(func(b *board) error {
		b.NextID = 5
		b.Schedules = []schedule{{ID: 1, Time: "08:00"}, {ID: 2, Interval: &scheduleInterval{Minutes: 1}}}
		b.Cards = []card{
			{ID: 3, Status: done, Source: "定期 #2 2026-11-23 10:00", Result: "実行結果"},
			{ID: 4, Status: running, ScheduleID: 2},
			{ID: 5, Status: todo},
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	u := newBoardUI(s, "true")
	u.refresh()
	press := func(k tcell.Key) { u.key(tcell.NewEventKey(k, 0, 0)) }
	key := func(r rune) { u.key(tcell.NewEventKey(tcell.KeyRune, r, 0)) }
	// 一覧の選択とは異なるカードを、番号指定で開く。
	u.execute("result 3")
	key('t')
	if u.activePage() != "input" || u.input.GetText() != "1m" {
		t.Fatal("古いカードの履歴から元の予定を選べません")
	}
	u.input.SetText("15m")
	press(tcell.KeyEnter)
	if text := u.detail.GetText(false); u.activePage() != "detail" || !strings.Contains(text, "15分おき") || !strings.Contains(text, "実行結果") {
		t.Fatalf("保存後のタスク詳細: %s", text)
	}
	press(tcell.KeyEscape)
	for i := todoLane; i < workLane; i++ {
		press(tcell.KeyRight)
	}
	key('v')
	key('t')
	u.input.SetText("30m")
	press(tcell.KeyEscape)
	if text := u.detail.GetText(false); u.activePage() != "detail" || !strings.Contains(text, "#4") || !strings.Contains(text, "15分おき") {
		t.Fatalf("取り消し後のタスク詳細: %s", text)
	}
	if _, err := parseCommand("remove 2").run(s, "true", s.dir); err != nil {
		t.Fatal(err)
	}
	u.refresh()
	key('t')
	if u.activePage() != "detail" || !strings.Contains(u.detail.GetText(false), "解除済み") {
		t.Fatal("解除済みの予定が編集対象になりました")
	}
	u.execute("result 5")
	key('t')
	if u.activePage() != "detail" || strings.Contains(u.hint.GetText(false), "設定変更") {
		t.Fatal("手動タスクに定期実行の編集操作が表示されました")
	}

	press(tcell.KeyEscape)
	press(tcell.KeyBacktab)
	press(tcell.KeyEnter)
	key('t')
	if u.input.GetText() != "08:00" {
		t.Fatal("自動化タブの詳細から時刻を編集できません")
	}
}
