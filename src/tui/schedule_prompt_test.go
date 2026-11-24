package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/schedule"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/store"
	"github.com/gdamore/tcell/v2"
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
	if u.activePage() != "schedule-form" || u.scheduleForm.prompt.GetText() != "前の内容" {
		t.Fatal("現在の実行内容を編集できません")
	}
	u.scheduleForm.prompt.SetText("　 ", true)
	press(tcell.KeyCtrlS)
	if u.activePage() != "schedule-form" || !strings.Contains(u.notice.GetText(false), "実行内容を入力") {
		t.Fatal("空欄を保存できてしまいました")
	}
	u.scheduleForm.prompt.SetText("取り消す内容", true)
	press(tcell.KeyEscape)
	edit()
	if u.scheduleForm.prompt.GetText() != "前の内容" {
		t.Fatal("取り消した内容が残っています")
	}

	if err := s.Edit(func(b *board.State) error { b.Schedules[0].Prompt = "別ペインの内容"; return nil }); err != nil {
		t.Fatal(err)
	}
	u.scheduleForm.prompt.SetText("競合する変更", true)
	press(tcell.KeyCtrlS)
	if u.activePage() != "schedule-form" || u.scheduleForm.prompt.GetText() != "別ペインの内容" || !strings.Contains(u.notice.GetText(false), "別のペイン") {
		t.Fatal("競合した編集が最新の内容に戻りませんでした")
	}
	u.scheduleForm.prompt.SetText("  新しい内容\n\n  追加の手順  ", true)
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
	if err := setBoardScreen(u.app, screen); err != nil {
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
