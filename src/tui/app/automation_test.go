package app

import (
	"strings"
	"testing"
	"time"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/application/command"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/schedule"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/store"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/tui/terminal"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestEditSchedulePromptFromDetail(t *testing.T) {
	s := store.Store{Dir: t.TempDir()}
	nextRun := time.Now().UTC().Truncate(time.Second)
	if err := s.Edit(func(b *board.State) error {
		id := b.AddSchedule(schedule.Rule{Dir: s.Dir, Prompt: "前の内容", Interval: &schedule.Interval{Minutes: 15, NextRun: nextRun}})
		b.NewCard("前の内容", s.Dir, "定期 #1").ScheduleID = id
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	u := newBoardUI(s, "")
	u.refresh()
	press := func(key tcell.Key) { u.key(tcell.NewEventKey(key, 0, 0)) }
	edit := func() { u.key(tcell.NewEventKey(tcell.KeyRune, 'e', 0)) }
	press(tcell.KeyBacktab)
	press(tcell.KeyEnter)
	if !strings.Contains(u.hint.GetText(false), "e 予定を編集") {
		t.Fatal("予定の詳細に編集操作が表示されていません")
	}
	edit()
	if u.activePage() != "schedule-form" || schedulePromptField(u).GetText() != "前の内容" {
		t.Fatal("現在の実行内容を編集できません")
	}
	schedulePromptField(u).SetText("　 ", true)
	press(tcell.KeyCtrlS)
	if u.activePage() != "schedule-form" || !strings.Contains(u.notice.GetText(false), "実行内容を入力") {
		t.Fatal("空欄を保存できてしまいました")
	}
	schedulePromptField(u).SetText("取り消す内容", true)
	press(tcell.KeyEscape)
	edit()
	if schedulePromptField(u).GetText() != "前の内容" {
		t.Fatal("取り消した内容が残っています")
	}

	if err := s.Edit(func(b *board.State) error { b.Schedules[0].Prompt = "別ペインの内容"; return nil }); err != nil {
		t.Fatal(err)
	}
	schedulePromptField(u).SetText("競合する変更", true)
	press(tcell.KeyCtrlS)
	if u.activePage() != "schedule-form" || schedulePromptField(u).GetText() != "別ペインの内容" || !strings.Contains(u.notice.GetText(false), "別のペイン") {
		t.Fatal("競合した編集が最新の内容に戻りませんでした")
	}
	schedulePromptField(u).SetText("  新しい内容\n\n  追加の手順  ", true)
	press(tcell.KeyCtrlS)
	if u.activePage() != "detail" || u.app.GetFocus() != u.detail || !strings.Contains(u.detail.GetText(false), "新しい内容\n\n  追加の手順") {
		t.Fatal("保存後の詳細に変更が反映されませんでした")
	}
	b, err := s.Read()
	if err != nil {
		t.Fatal(err)
	}
	r := b.Schedules[0]
	if r.Prompt != "新しい内容\n\n  追加の手順" || r.Interval.Minutes != 15 || !r.Interval.NextRun.Equal(nextRun) || b.Cards[0].Prompt != "前の内容" {
		t.Fatalf("保存後の予定と履歴 = %+v, %+v", r, b.Cards)
	}
	b.RecordDueSchedules(nextRun)
	if len(b.Cards) != 2 || b.Cards[1].Prompt != r.Prompt {
		t.Fatalf("次の実行カード = %+v", b.Cards)
	}
}

func TestScheduleFormPreservesPastedLines(t *testing.T) {
	s := store.Store{Dir: t.TempDir()}
	u := newBoardUI(s, "true")
	u.workDir = s.Dir
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := terminal.SetScreen(u.app, screen); err != nil {
		t.Fatal(err)
	}
	key := func(k tcell.Key) { screen.PostEventWait(tcell.NewEventKey(k, 0, 0)) }
	typeText := func(text string) {
		for _, r := range text {
			screen.PostEventWait(tcell.NewEventKey(tcell.KeyRune, r, 0))
		}
	}
	go func() {
		key(tcell.KeyBacktab)
		typeText("i15m")
		key(tcell.KeyTab)
		typeText("確認する")
		key(tcell.KeyEnter)
		screen.PostEventWait(tcell.NewEventPaste(true))
		typeText("報告する")
		key(tcell.KeyEnter)
		key(tcell.KeyCtrlJ)
		key(tcell.KeyCtrlJ)
		typeText("  補足")
		screen.PostEventWait(tcell.NewEventPaste(false))
		key(tcell.KeyTab)
		key(tcell.KeyEnter)
		key(tcell.KeyCtrlC)
	}()
	if err := u.app.Run(); err != nil {
		t.Fatal(err)
	}
	b, err := s.Read()
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Schedules) != 1 || b.Schedules[0].Prompt != "確認する\n報告する\n\n  補足" {
		t.Fatalf("フォームから保存した予定 = %+v", b.Schedules)
	}
}

func scheduleTimeField(u *boardUI) *tview.InputField {
	return u.scheduleForm.GetFormItem(0).(*tview.InputField)
}

func schedulePromptField(u *boardUI) *tview.TextArea {
	return u.scheduleForm.GetFormItemByLabel("実行内容").(*tview.TextArea)
}

func TestAutomationTabManagesDailySchedule(t *testing.T) {
	s := store.Store{Dir: t.TempDir()}
	t.Setenv("HOME", s.Dir)
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
	if u.activePage() != "schedule-form" || scheduleTimeField(u).GetLabel() != "毎日の時刻" {
		t.Fatal("作業先の選択後に定期実行を入力できません")
	}
	scheduleTimeField(u).SetText("09:00")
	schedulePromptField(u).SetText("テスト", true)
	press(tcell.KeyCtrlS)
	b, err := s.Read()
	if err != nil {
		t.Fatal(err)
	}
	r, ok := u.schedules.Selected()
	if !ok || r.Prompt != "テスト" || r.Dir != s.Dir || r.Time != "09:00" || len(b.Cards) != 0 {
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
	b, err = s.Read()
	if err != nil {
		t.Fatal(err)
	}
	if u.schedules.GetItemCount() != 0 || len(b.Schedules) != 0 {
		t.Fatal("確定した予定を解除できません")
	}
	for _, page := range []string{"tasks", "agents", "reviews", "automation"} {
		press(tcell.KeyTab)
		if u.activePage() != page {
			t.Fatalf("タブ切替後のページ = %q, 期待値 = %q", u.activePage(), page)
		}
	}
}

func TestScheduleSelectionAfterAnotherPaneRemovesRule(t *testing.T) {
	s := store.Store{Dir: t.TempDir()}
	if err := s.Edit(func(b *board.State) error {
		b.Schedules = []schedule.Rule{{ID: 1}, {ID: 2}, {ID: 3}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	u := newBoardUI(s, "true")
	u.refresh()
	u.key(tcell.NewEventKey(tcell.KeyBacktab, 0, 0))
	u.schedules.SetCurrentItem(1)
	if _, err := command.Parse("remove 1").Run(s, "true", s.Dir); err != nil {
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
	s := store.Store{Dir: t.TempDir()}
	if err := s.RegisterProject(s.Dir); err != nil {
		t.Fatal(err)
	}
	u := newBoardUI(s, "true")
	u.refresh()
	u.key(tcell.NewEventKey(tcell.KeyBacktab, 0, 0))
	u.key(tcell.NewEventKey(tcell.KeyRune, 'i', 0))
	if u.activePage() != "schedule-form" || scheduleTimeField(u).GetLabel() != "実行間隔" {
		t.Fatal("間隔指定の入力を開始できません")
	}
	scheduleTimeField(u).SetText("5m")
	schedulePromptField(u).SetText("変更を確認する", true)
	u.key(tcell.NewEventKey(tcell.KeyCtrlS, 0, 0))
	if u.schedules.GetItemCount() != 1 {
		t.Fatal("登録した予定が表示されません")
	}
	text, _ := u.schedules.GetItemText(0)
	if text != "5分おき  変更を確認する" {
		t.Fatalf("一覧の予定: %s", text)
	}
}

func TestEditIntervalFromAutomationTab(t *testing.T) {
	s := store.Store{Dir: t.TempDir()}
	original := time.Now().Add(time.Minute)
	if err := s.Edit(func(b *board.State) error {
		b.NextID = 3
		b.Cards = []board.Card{{ID: 3, Prompt: "履歴", Status: board.Done}}
		b.Schedules = []schedule.Rule{
			{ID: 1, Time: "09:00"},
			{ID: 2, Prompt: "差分の確認", Dir: s.Dir, Interval: &schedule.Interval{Minutes: 1, NextRun: original}},
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
	if _, err := command.Parse("remove 1").Run(s, "true", s.Dir); err != nil {
		t.Fatal(err)
	}
	u.refresh()
	before := time.Now()
	u.input.SetText("15m")
	press(tcell.KeyEnter)
	b, err := s.Read()
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Schedules) != 1 || len(b.Cards) != 1 || b.Cards[0].ID != 3 || b.NextID != 3 {
		t.Fatalf("更新後の履歴: %+v", b)
	}
	r := b.Schedules[0]
	if u.activePage() != "automation" || r.ID != 2 || r.Prompt != "差分の確認" || r.Dir != s.Dir || r.Interval.Minutes != 15 {
		t.Fatalf("更新後の予定: %+v", r)
	}
	if r.Interval.NextRun.Before(before.Add(15*time.Minute)) || r.Interval.NextRun.After(time.Now().Add(15*time.Minute)) || r.Due(original) {
		t.Fatalf("次回の実行時刻: %v", r.Interval.NextRun)
	}
	edit()
	press(tcell.KeyEnter)
	same, ok := u.schedules.Selected()
	if !ok || !same.Interval.NextRun.Equal(r.Interval.NextRun) {
		t.Fatal("同じ間隔で確定すると実行時刻がずれました")
	}
}

func TestTimingFromTaskDetail(t *testing.T) {
	s := store.Store{Dir: t.TempDir()}
	if err := s.Edit(func(b *board.State) error {
		b.NextID = 5
		b.Schedules = []schedule.Rule{{ID: 1, Time: "08:00"}, {ID: 2, Interval: &schedule.Interval{Minutes: 1}}}
		b.Cards = []board.Card{
			{ID: 3, Status: board.Done, Source: "定期 #2 2026-11-23 10:00", Result: "実行結果"},
			{ID: 4, Status: board.Running, ScheduleID: 2},
			{ID: 5, Status: board.Todo},
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
	press(tcell.KeyBacktab)
	key('h')
	key('v')
	key('t')
	u.input.SetText("30m")
	press(tcell.KeyEscape)
	if text := u.detail.GetText(false); u.activePage() != "detail" || !strings.Contains(text, "#4") || !strings.Contains(text, "15分おき") {
		t.Fatalf("取り消し後のタスク詳細: %s", text)
	}
	if _, err := command.Parse("remove 2").Run(s, "true", s.Dir); err != nil {
		t.Fatal(err)
	}
	u.refresh()
	key('t')
	if u.activePage() != "detail" || !strings.Contains(u.detail.GetText(false), "解除済み") {
		t.Fatal("解除済みの予定が編集対象になりました")
	}
	if hint := u.hint.GetText(false); !strings.Contains(hint, "x キャンセル") || strings.Contains(hint, "t 定期実行の設定変更") {
		t.Fatalf("予定解除後の実行中カードの操作案内: %s", hint)
	}
	key('x')
	if u.detail.Card().Status != board.Canceled || strings.Contains(u.hint.GetText(false), "x キャンセル") {
		t.Fatal("予定解除後のカードにキャンセルが反映されませんでした")
	}
	u.execute("result 5")
	key('t')
	if u.activePage() != "detail" || strings.Contains(u.hint.GetText(false), "設定変更") {
		t.Fatal("手動タスクに定期実行の編集操作が表示されました")
	}

	press(tcell.KeyEscape)
	key('h')
	press(tcell.KeyEnter)
	key('t')
	if u.input.GetText() != "08:00" {
		t.Fatal("自動化タブの詳細から時刻を編集できません")
	}
	u.input.SetText(" 10:30 ")
	press(tcell.KeyEnter)
	daily, ok := u.schedules.ByID(1)
	if !ok || daily.Time != "10:30" || daily.Interval != nil || u.activePage() != "detail" {
		t.Fatalf("時刻変更後の毎日の予定: %+v", daily)
	}
}

func TestTimingConflictReloadsCurrentSetting(t *testing.T) {
	for _, tc := range []struct {
		name                string
		rule                schedule.Rule
		newValue, nextValue string
	}{
		{"毎日", schedule.Rule{Time: "09:00"}, "10:30", "11:00"},
		{"間隔", schedule.Rule{Interval: &schedule.Interval{Minutes: 15}}, "30m", "45m"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := store.Store{Dir: t.TempDir()}
			if err := s.Edit(func(b *board.State) error { b.AddSchedule(tc.rule); return nil }); err != nil {
				t.Fatal(err)
			}
			left, right := newBoardUI(s, "true"), newBoardUI(s, "true")
			for _, u := range []*boardUI{left, right} {
				u.refresh()
				u.key(tcell.NewEventKey(tcell.KeyBacktab, 0, 0))
				u.key(tcell.NewEventKey(tcell.KeyRune, 't', 0))
			}
			enter := tcell.NewEventKey(tcell.KeyEnter, 0, 0)
			right.input.SetText(tc.newValue)
			right.key(enter)
			left.key(enter)
			b, err := s.Read()
			if err != nil || b.Schedules[0].TimingValue() != tc.newValue {
				t.Fatalf("競合後の保存内容: %+v, %v", b, err)
			}
			if left.activePage() != "input" || left.input.GetText() != tc.newValue || !strings.Contains(left.notice.GetText(false), "別のペイン") {
				t.Fatalf("競合後の入力欄 = %q, 通知 = %q", left.input.GetText(), left.notice.GetText(false))
			}
			// 実行履歴だけが更新されても、最新の設定を見てやり直した編集は保存できる。
			if err := s.Edit(func(b *board.State) error {
				r := &b.Schedules[0]
				if r.Interval != nil {
					r.Interval.NextRun = r.Interval.NextRun.Add(time.Hour)
				} else {
					r.LastDate = time.Now().Format("2006-01-02")
				}
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			left.input.SetText(tc.nextValue)
			left.key(enter)
			b, err = s.Read()
			if err != nil || b.Schedules[0].TimingValue() != tc.nextValue || left.activePage() != "automation" {
				t.Fatalf("競合解消後の保存: %+v, %v", b, err)
			}
		})
	}
}
