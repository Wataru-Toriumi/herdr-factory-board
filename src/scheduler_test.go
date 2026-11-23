package main

import (
	"testing"
	"time"
)

func TestSchedulerLockLifecycle(t *testing.T) {
	s := store{t.TempDir()}
	checkRunning := func(want bool) {
		t.Helper()
		active, err := schedulerRunning(s)
		if err != nil || active != want {
			t.Fatalf("稼働状態 = %v, %v; want %v", active, err, want)
		}
	}
	checkRunning(false)
	lock, err := tryAcquireSchedulerLock(s)
	if err != nil || lock == nil {
		t.Fatalf("初回のロック取得: %v", err)
	}
	checkRunning(true)
	other, err := tryAcquireSchedulerLock(s)
	if err != nil || other != nil {
		t.Fatalf("二重起動時のロック = %v, %v", other, err)
	}
	if err := lock.Close(); err != nil {
		t.Fatal(err)
	}
	checkRunning(false)
}

func TestDailyRegistrationFirstRun(t *testing.T) {
	registered := time.Date(2026, 11, 23, 9, 0, 30, 0, time.Local)
	for _, tc := range []struct {
		clock string
		first time.Time
	}{
		{"08:59", time.Date(2026, 11, 24, 8, 59, 0, 0, time.Local)},
		{"09:00", registered},
		{"09:01", time.Date(2026, 11, 23, 9, 1, 0, 0, time.Local)},
	} {
		t.Run(tc.clock, func(t *testing.T) {
			var r schedule
			if err := r.setTiming("daily", tc.clock, registered); err != nil {
				t.Fatal(err)
			}
			if tc.first.After(registered) && due(r, registered) {
				t.Fatal("登録直後に実行対象になりました")
			}
			if !due(r, tc.first) {
				t.Fatalf("初回実行時刻 %v に実行対象になりませんでした", tc.first)
			}
		})
	}
}

func TestIntervalScheduleAcrossTicks(t *testing.T) {
	s := store{t.TempDir()}
	if err := registerProject(s, s.dir); err != nil {
		t.Fatal(err)
	}
	before := time.Now()
	if _, err := command(s, "true", s.dir, "every 15m 差分を確認する"); err != nil {
		t.Fatal(err)
	}
	b, err := s.read()
	if err != nil {
		t.Fatal(err)
	}
	r := b.Schedules[0]
	if r.Interval == nil || r.Interval.Minutes != 15 || r.frequency() != "15分おき" {
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
		if err := tick(s, "true", first.Add(tc.offset)); err != nil {
			t.Fatal(err)
		}
		b, err = s.read()
		if err != nil {
			t.Fatal(err)
		}
		if len(b.Cards) != tc.cards || !b.Schedules[0].Interval.NextRun.Equal(first.Add(tc.next)) {
			t.Fatalf("初回から %v 後の状態: %+v", tc.offset, b)
		}
	}
}

func TestIntervalCommandRejectsInvalidInput(t *testing.T) {
	s := store{t.TempDir()}
	if err := registerProject(s, s.dir); err != nil {
		t.Fatal(err)
	}
	for _, args := range []string{"0m 確認", "-1m 確認", "30s 確認", "1m30s 確認", "15 確認", "15m"} {
		if _, err := command(s, "true", s.dir, "every "+args); err == nil {
			t.Errorf("無効な入力が受理されました: %q", args)
		}
	}
	b, err := s.read()
	if err != nil || len(b.Schedules) != 0 {
		t.Fatalf("入力エラー後の予定: %+v, %v", b.Schedules, err)
	}
}

func TestEditDailyTime(t *testing.T) {
	now := time.Date(2026, 11, 23, 10, 0, 0, 0, time.Local)
	for _, tc := range []struct {
		name, lastDate, when string
		dueToday             bool
	}{
		{"これからの時刻へ変更", "", "11:00", true},
		{"過ぎた時刻へ変更", "", "09:30", false},
		{"今日の分を実行済み", "2026-11-23", "11:00", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := schedule{Time: "12:00", LastDate: tc.lastDate}
			if err := r.setTiming("daily", tc.when, now); err != nil {
				t.Fatal(err)
			}
			if r.Time != tc.when || due(r, now.Add(2*time.Hour)) != tc.dueToday || !due(r, now.AddDate(0, 0, 1).Add(2*time.Hour)) {
				t.Fatalf("変更後の予定: %+v", r)
			}
		})
	}
}
