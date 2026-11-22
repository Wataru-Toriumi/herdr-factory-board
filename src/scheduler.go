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

func registerDaily(s store, herdr, arg string) error {
	clock, prompt, _ := strings.Cut(arg, " ")
	parsed, err := time.Parse("15:04", clock)
	if err != nil || parsed.Format("15:04") != clock {
		return errors.New("daily HH:MM 作業内容 の形式で指定してください")
	}
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return errors.New("作業内容を入力してください")
	}
	if err := s.edit(func(b *board) error {
		if b.Dir == "" {
			return errors.New("先に dir で作業先を指定してください")
		}
		b.addDailySchedule(clock, prompt, time.Now())
		return nil
	}); err != nil {
		return err
	}
	return ensureScheduler(s, herdr)
}

func (b *board) addDailySchedule(clock, prompt string, now time.Time) {
	b.NextID++
	r := schedule{ID: b.NextID, Time: clock, Prompt: prompt, Dir: b.Dir}
	if clock < now.Format("15:04") {
		r.LastDate = now.Format("2006-01-02")
	}
	b.Schedules = append(b.Schedules, r)
}

func ensureScheduler(s store, herdr string) error {
	active, err := schedulerRunning(s)
	if err != nil || active {
		return err
	}
	return openPlugin(herdr, "scheduler")
}

func due(rule schedule, now time.Time) bool {
	return rule.LastDate < now.Format("2006-01-02") && rule.Time <= now.Format("15:04")
}

// 同じ日の二重実行を防ぐため、実行日の更新と履歴カードの作成を一度の書き込みで行う。
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
					r.LastDate = now.Format("2006-01-02")
					source := fmt.Sprintf("定期 #%d %s %s", r.ID, r.LastDate, r.Time)
					current.newCard(r.Prompt, r.Dir, source)
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
