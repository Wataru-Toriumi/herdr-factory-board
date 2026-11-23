// Package schedule は定期実行の時刻と実行履歴を管理する。
package schedule

import (
	"errors"
	"fmt"
	"time"
)

// Rule はボードに保存する定期実行の予定を表す。
type Rule struct {
	ID                          int
	Time, Prompt, Dir, LastDate string
	Interval                    *Interval `json:",omitempty"`
}

// IndexByID は指定した予定番号の位置を返す。見つからなければ -1 を返す。
func IndexByID(rules []Rule, id int) int {
	for i := range rules {
		if rules[i].ID == id {
			return i
		}
	}
	return -1
}

// Interval は分単位の実行間隔と次の予定時刻を保持する。
type Interval struct {
	Minutes int64
	NextRun time.Time
}

// SetTiming は毎日の実行時刻または実行間隔を設定する。
func (r *Rule) SetTiming(kind, when string, now time.Time) error {
	if kind == "every" {
		period, err := time.ParseDuration(when)
		if err != nil || period < time.Minute || period%time.Minute != 0 {
			return errors.New("間隔は 15m のように1分単位で指定してください")
		}
		if r.Interval == nil || r.Interval.period() != period {
			r.Interval = &Interval{Minutes: int64(period / time.Minute), NextRun: now.Add(period)}
		}
		return nil
	}

	parsed, err := time.Parse("15:04", when)
	if err != nil || parsed.Format("15:04") != when {
		return errors.New("時刻は HH:MM の形式で指定してください")
	}
	if r.Time != when {
		r.Time = when
		if when < now.Format("15:04") {
			r.LastDate = now.Format("2006-01-02")
		}
	}
	return nil
}

func (i Interval) period() time.Duration {
	return time.Duration(i.Minutes) * time.Minute
}

// Frequency はボードに表示する頻度を返す。
func (r Rule) Frequency() string {
	if r.Interval != nil {
		return fmt.Sprintf("%d分おき", r.Interval.Minutes)
	}
	return "毎日 " + r.Time
}

// Due は現在時刻までに未実行の予定があるかを返す。
func (r Rule) Due(now time.Time) bool {
	if i := r.Interval; i != nil {
		return i.period() > 0 && !i.NextRun.IsZero() && !now.Before(i.NextRun)
	}
	return r.LastDate < now.Format("2006-01-02") && r.Time <= now.Format("15:04")
}

// RecordOccurrence は Due が成立した予定の実行を記録し、履歴カードの作成元を返す。
func (r *Rule) RecordOccurrence(now time.Time) string {
	if i := r.Interval; i != nil {
		// 休止中に複数回分を過ぎた場合も、登録時刻を基準に直近の一回を記録する。
		period := i.period()
		scheduled := now.Add(-(now.Sub(i.NextRun) % period))
		i.NextRun = scheduled.Add(period)
		return fmt.Sprintf("定期 #%d %s", r.ID, scheduled.Format("2006-01-02 15:04"))
	}
	r.LastDate = now.Format("2006-01-02")
	return fmt.Sprintf("定期 #%d %s %s", r.ID, r.LastDate, r.Time)
}
