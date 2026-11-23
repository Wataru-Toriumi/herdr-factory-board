package command

import (
	"testing"
	"time"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/application/scheduler"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/store"
)

func TestCardsAndSchedulesShareIDs(t *testing.T) {
	s := store.Store{Dir: t.TempDir()}
	for _, line := range []string{"add 手動タスク", "every 15m 定期タスク"} {
		if _, err := Parse(line).Run(s, "true", s.Dir); err != nil {
			t.Fatal(err)
		}
	}
	if err := scheduler.RunDue(s, "true", time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	b, err := s.Read()
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Cards) != 2 || len(b.Schedules) != 1 {
		t.Fatalf("保存されたカードと予定の件数 = %d, %d", len(b.Cards), len(b.Schedules))
	}
	manual, rule, history := b.Cards[0], b.Schedules[0], b.Cards[1]
	if manual.ID != 1 || rule.ID != 2 || history.ID != 3 || b.NextID != 3 || history.ScheduleID != rule.ID {
		t.Fatalf("手動カード #%d、予定 #%d、履歴カード #%d（予定 #%d）、採番済み %d",
			manual.ID, rule.ID, history.ID, history.ScheduleID, b.NextID)
	}
}

func TestIntervalScheduleAcrossTicks(t *testing.T) {
	s := store.Store{Dir: t.TempDir()}
	if err := s.RegisterProject(s.Dir); err != nil {
		t.Fatal(err)
	}
	before := time.Now()
	if _, err := Parse("every 15m 差分を確認する").Run(s, "true", s.Dir); err != nil {
		t.Fatal(err)
	}
	b, err := s.Read()
	if err != nil {
		t.Fatal(err)
	}
	r := b.Schedules[0]
	if r.Interval == nil || r.Interval.Minutes != 15 || r.Frequency() != "15分おき" {
		t.Fatalf("登録した予定: %+v", r)
	}
	first := r.Interval.NextRun
	if first.Before(before.Add(15*time.Minute)) || first.After(time.Now().Add(15*time.Minute)) {
		t.Fatalf("初回予定: %v", first)
	}
	for _, tc := range []struct {
		offset time.Duration
		cards  int
		next   time.Duration
	}{
		{-time.Nanosecond, 0, 0},
		{0, 1, 15 * time.Minute},
		{0, 1, 15 * time.Minute},
		{15*time.Minute + 3*time.Second, 2, 30 * time.Minute},
		{26*time.Hour + 2*time.Minute, 3, 26*time.Hour + 15*time.Minute},
	} {
		if err := scheduler.RunDue(s, "true", first.Add(tc.offset)); err != nil {
			t.Fatal(err)
		}
		b, err = s.Read()
		if err != nil {
			t.Fatal(err)
		}
		if len(b.Cards) != tc.cards || !b.Schedules[0].Interval.NextRun.Equal(first.Add(tc.next)) {
			t.Fatalf("初回から %v 後の状態: %+v", tc.offset, b)
		}
	}
}

func TestIntervalCommandRejectsInvalidInput(t *testing.T) {
	s := store.Store{Dir: t.TempDir()}
	if err := s.RegisterProject(s.Dir); err != nil {
		t.Fatal(err)
	}
	for _, args := range []string{"0m 確認", "-1m 確認", "30s 確認", "1m30s 確認", "15 確認", "15m"} {
		if _, err := Parse("every "+args).Run(s, "true", s.Dir); err == nil {
			t.Errorf("無効な入力が受理されました: %q", args)
		}
	}
	b, err := s.Read()
	if err != nil || len(b.Schedules) != 0 || b.NextID != 0 {
		t.Fatalf("入力エラー後の予定 = %+v, 採番済み = %d, エラー = %v", b.Schedules, b.NextID, err)
	}
}
