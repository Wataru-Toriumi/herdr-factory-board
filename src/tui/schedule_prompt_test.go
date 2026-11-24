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
	if !strings.Contains(u.hint.GetText(false), "e 実行内容を編集") {
		t.Fatal("予定の詳細に編集操作が表示されていません")
	}
	edit()
	if u.activePage() != "input" || u.input.GetText() != "前の内容" {
		t.Fatal("現在の実行内容を編集できません")
	}
	u.input.SetText("　 ")
	press(tcell.KeyEnter)
	if u.activePage() != "input" || !strings.Contains(u.notice.GetText(false), "実行内容を入力") {
		t.Fatal("空欄を保存できてしまいました")
	}
	u.input.SetText("取り消す内容")
	press(tcell.KeyEscape)
	edit()
	if u.input.GetText() != "前の内容" {
		t.Fatal("取り消した内容が残っています")
	}

	if err := s.Edit(func(b *board.State) error { b.Schedules[0].Prompt = "別ペインの内容"; return nil }); err != nil {
		t.Fatal(err)
	}
	u.input.SetText("競合する変更")
	press(tcell.KeyEnter)
	if u.activePage() != "input" || u.input.GetText() != "別ペインの内容" || !strings.Contains(u.notice.GetText(false), "別のペイン") {
		t.Fatal("競合した編集が最新の内容に戻りませんでした")
	}
	u.input.SetText("  新しい内容  ")
	press(tcell.KeyEnter)
	if u.activePage() != "detail" || u.app.GetFocus() != u.detail || !strings.Contains(u.detail.GetText(false), "新しい内容") {
		t.Fatal("保存後の詳細に変更が反映されませんでした")
	}
	b, err := s.Read()
	if err != nil {
		t.Fatal(err)
	}
	r := b.Schedules[0]
	if r.Prompt != "新しい内容" || r.Interval.Minutes != 15 || !r.Interval.NextRun.Equal(nextRun) || b.Cards[0].Prompt != "前の内容" {
		t.Fatalf("保存後の予定と履歴 = %+v, %+v", r, b.Cards)
	}
	b.RecordDueSchedules(nextRun)
	if len(b.Cards) != 2 || b.Cards[1].Prompt != r.Prompt {
		t.Fatalf("次の実行カード = %+v", b.Cards)
	}
}
