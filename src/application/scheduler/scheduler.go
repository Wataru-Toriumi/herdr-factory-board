// Package scheduler は定期実行の予定を管理し、実行時刻を迎えたカードを起動する。
package scheduler

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/application/worker"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/schedule"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/herdr"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/store"
)

// 稼働確認と起動を直列化し、確認中の一時的なロックを別のスケジューラーの稼働と誤認するのを防ぐ。
func schedulerLease(s store.Store, probe bool) (lease *os.File, active bool, err error) {
	gate, err := s.Lock("scheduler-start.lock", false)
	if err != nil {
		return nil, false, err
	}
	defer gate.Close()
	l, err := s.Lock("scheduler.lock", true)
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

func Running(s store.Store) (bool, error) {
	_, active, err := schedulerLease(s, true)
	return active, err
}

// すでに稼働中なら nil を返す。取得したロックは終了時に閉じる。
func tryAcquireSchedulerLock(s store.Store) (*os.File, error) {
	lock, _, err := schedulerLease(s, false)
	return lock, err
}

func Register(s store.Store, herdrBin, dir, kind, arg string) error {
	when, prompt, _ := strings.Cut(arg, " ")
	rule := schedule.Rule{Prompt: strings.TrimSpace(prompt)}
	if rule.Prompt == "" {
		return errors.New("作業内容を入力してください")
	}
	if err := s.Edit(func(b *board.State) error {
		if dir == "" {
			return errors.New("先に dir で作業先を指定してください")
		}
		if err := rule.SetTiming(kind, when, time.Now()); err != nil {
			return err
		}
		b.NextID++
		rule.ID, rule.Dir = b.NextID, dir
		b.Schedules = append(b.Schedules, rule)
		return nil
	}); err != nil {
		return err
	}
	return EnsureRunning(s, herdrBin)
}

func UpdateTiming(s store.Store, id int, value string) error {
	return s.Edit(func(b *board.State) error {
		i := schedule.IndexByID(b.Schedules, id)
		if i < 0 {
			return fmt.Errorf("定期実行 #%d がありません", id)
		}
		r := &b.Schedules[i]
		return r.SetTiming(r.TimingKind(), strings.TrimSpace(value), time.Now())
	})
}

func Remove(s store.Store, id int) error {
	return s.Edit(func(b *board.State) error {
		i := schedule.IndexByID(b.Schedules, id)
		if i < 0 {
			return fmt.Errorf("定期実行 #%d がありません", id)
		}
		b.Schedules = append(b.Schedules[:i], b.Schedules[i+1:]...)
		return nil
	})
}

func EnsureRunning(s store.Store, herdrBin string) error {
	active, err := Running(s)
	if err != nil || active {
		return err
	}
	return herdr.OpenPlugin(herdrBin, "scheduler")
}

// 同じ予定の二重実行を防ぐため、実行時刻の更新と履歴カードの作成を一度に保存する。
func RunDue(s store.Store, herdrBin string, now time.Time) error {
	b, err := s.Read()
	if err != nil {
		return err
	}
	// 実行時刻を迎えた予定があるときだけ書き込む。
	// 読み込み後に予定が変更されることがあるため、ロックを取得してから再確認する。
	if b.HasDueSchedule(now) {
		err = s.Edit(func(current *board.State) error {
			current.RecordDueSchedules(now)
			b = *current
			return nil
		})
		if err != nil {
			return err
		}
	}
	// 前回の停止直前に保存した未実行の定期カードも起動する。
	for _, c := range b.Cards {
		if c.Status == board.Todo && c.Source != "" {
			if launchErr := worker.Start(s, herdrBin, c.ID); launchErr != nil {
				err = errors.Join(err, launchErr)
			}
		}
	}
	return err
}

func Run(s store.Store, herdrBin string) error {
	lock, err := tryAcquireSchedulerLock(s)
	if err != nil || lock == nil {
		return err
	}
	defer lock.Close()
	fmt.Println("Factory Board 定期実行中。このタブを閉じると停止します。")
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		if err := RunDue(s, herdrBin, time.Now()); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		<-ticker.C
	}
}
