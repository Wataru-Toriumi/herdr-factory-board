package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"
)

// 稼働確認と起動を直列化し、確認中の一時的なロックを別のスケジューラーの稼働と誤認するのを防ぐ。
func schedulerLease(s store, probe bool) (lease *os.File, active bool, err error) {
	gate, err := s.lock("scheduler-start.lock", false)
	if err != nil {
		return nil, false, err
	}
	defer gate.Close()
	l, err := s.lock("scheduler.lock", true)
	if errors.Is(err, syscall.EWOULDBLOCK) {
		return nil, true, nil
	}
	if err != nil {
		return nil, false, err
	}
	if probe {
		return nil, false, l.Close()
	}
	return l, false, nil
}

func schedulerRunning(s store) (bool, error) {
	_, active, err := schedulerLease(s, true)
	return active, err
}

// すでに稼働中なら nil を返す。取得したロックは終了時に閉じる。
func tryAcquireSchedulerLock(s store) (*os.File, error) {
	lock, _, err := schedulerLease(s, false)
	return lock, err
}

func registerSchedule(s store, herdr, kind, arg string) error {
	when, prompt, _ := strings.Cut(arg, " ")
	rule := schedule{Prompt: strings.TrimSpace(prompt)}
	if rule.Prompt == "" {
		return errors.New("作業内容を入力してください")
	}
	if err := s.edit(func(b *board) error {
		if b.Dir == "" {
			return errors.New("先に dir で作業先を指定してください")
		}
		if err := rule.setTiming(kind, when, time.Now()); err != nil {
			return err
		}
		b.NextID++
		rule.ID, rule.Dir = b.NextID, b.Dir
		b.Schedules = append(b.Schedules, rule)
		return nil
	}); err != nil {
		return err
	}
	return ensureScheduler(s, herdr)
}

func (r *schedule) setTiming(kind, when string, now time.Time) error {
	if kind == "every" {
		period, err := time.ParseDuration(when)
		if err != nil || period < time.Minute || period%time.Minute != 0 {
			return errors.New("間隔は 15m のように1分単位で指定してください")
		}
		if r.Interval == nil || r.Interval.period() != period {
			r.Interval = &scheduleInterval{Minutes: int64(period / time.Minute), NextRun: now.Add(period)}
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

func updateScheduleTiming(s store, id int, value string) error {
	return s.edit(func(b *board) error {
		for i := range b.Schedules {
			r := &b.Schedules[i]
			if r.ID == id {
				kind := "daily"
				if r.Interval != nil {
					kind = "every"
				}
				return r.setTiming(kind, strings.TrimSpace(value), time.Now())
			}
		}
		return fmt.Errorf("定期実行 #%d がありません", id)
	})
}

func (i scheduleInterval) period() time.Duration {
	return time.Duration(i.Minutes) * time.Minute
}

func (r schedule) frequency() string {
	if r.Interval != nil {
		return fmt.Sprintf("%d分おき", r.Interval.Minutes)
	}
	return "毎日 " + r.Time
}

func ensureScheduler(s store, herdr string) error {
	active, err := schedulerRunning(s)
	if err != nil || active {
		return err
	}
	return openPlugin(herdr, "scheduler")
}

func due(rule schedule, now time.Time) bool {
	if i := rule.Interval; i != nil {
		return i.period() > 0 && !i.NextRun.IsZero() && !now.Before(i.NextRun)
	}
	return rule.LastDate < now.Format("2006-01-02") && rule.Time <= now.Format("15:04")
}

func (r *schedule) recordOccurrence(now time.Time) string {
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

// 同じ予定の二重実行を防ぐため、実行時刻の更新と履歴カードの作成を一度に保存する。
func tick(s store, herdr string, now time.Time) error {
	b, err := s.read()
	if err != nil {
		return err
	}
	// 実行時刻を迎えた予定があるときだけ書き込む。
	// 読み込み後に予定が変更されることがあるため、ロックを取得してから再確認する。
	for _, rule := range b.Schedules {
		if !due(rule, now) {
			continue
		}
		err = s.edit(func(current *board) error {
			for i := range current.Schedules {
				r := &current.Schedules[i]
				if due(*r, now) {
					source := r.recordOccurrence(now)
					current.newCard(r.Prompt, r.Dir, source).ScheduleID = r.ID
				}
			}
			b = *current
			return nil
		})
		if err != nil {
			return err
		}
		break
	}
	// 前回の停止直前に保存した未実行の定期カードも起動する。
	for _, c := range b.Cards {
		if c.Status == todo && c.Source != "" {
			if launchErr := startCard(s, herdr, c.ID); launchErr != nil {
				err = errors.Join(err, launchErr)
			}
		}
	}
	return err
}

func runScheduler(s store, herdr string) error {
	lock, err := tryAcquireSchedulerLock(s)
	if err != nil || lock == nil {
		return err
	}
	defer lock.Close()
	fmt.Println("Factory Board 定期実行中。このタブを閉じると停止します。")
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		if err := tick(s, herdr, time.Now()); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		<-ticker.C
	}
}
