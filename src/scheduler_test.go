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
			var b board
			b.addDailySchedule(tc.clock, "定期タスク", registered)
			r := b.Schedules[0]
			if tc.first.After(registered) && due(r, registered) {
				t.Fatal("登録直後に実行対象になりました")
			}
			if !due(r, tc.first) {
				t.Fatalf("初回実行時刻 %v に実行対象になりませんでした", tc.first)
			}
		})
	}
}
