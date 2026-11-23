package schedule

import (
	"testing"
	"time"
)

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
			var r Rule
			if err := r.SetTiming("daily", tc.clock, registered); err != nil {
				t.Fatal(err)
			}
			if tc.first.After(registered) && r.Due(registered) {
				t.Fatal("登録直後に実行対象になりました")
			}
			if !r.Due(tc.first) {
				t.Fatalf("初回実行時刻 %v に実行対象になりませんでした", tc.first)
			}
		})
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
			r := Rule{Time: "12:00", LastDate: tc.lastDate}
			if err := r.SetTiming("daily", tc.when, now); err != nil {
				t.Fatal(err)
			}
			if r.Time != tc.when || r.Due(now.Add(2*time.Hour)) != tc.dueToday || !r.Due(now.AddDate(0, 0, 1).Add(2*time.Hour)) {
				t.Fatalf("変更後の予定: %+v", r)
			}
		})
	}
}
