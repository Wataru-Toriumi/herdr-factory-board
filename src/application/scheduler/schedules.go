package scheduler

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/schedule"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/store"
)

func Register(s store.Store, herdrBin, dir, kind, arg, agent string) error {
	when, prompt, _ := strings.Cut(arg, " ")
	rule := schedule.Rule{Agent: agent, Prompt: strings.TrimSpace(prompt), Dir: dir}
	if rule.Prompt == "" {
		return errors.New("作業内容を入力してください")
	}
	var id int
	if err := s.Edit(func(b *board.State) error {
		if dir == "" {
			return errors.New("先に dir で作業先を指定してください")
		}
		if err := rule.SetTiming(kind, when, time.Now()); err != nil {
			return err
		}
		id = b.AddSchedule(rule)
		return nil
	}); err != nil {
		return err
	}
	if err := EnsureRunning(s, herdrBin); err != nil {
		return fmt.Errorf("予定 #%d は保存済みです。scheduler で起動を再試行してください（%w）", id, err)
	}
	return nil
}

func UpdateTiming(s store.Store, id int, value string) error {
	return editSchedule(s, id, func(b *board.State, i int) error {
		r := &b.Schedules[i]
		return r.SetTiming(r.TimingKind(), strings.TrimSpace(value), time.Now())
	})
}

func ChangeAgent(s store.Store, id int, seen, agent string) error {
	return editSchedule(s, id, func(b *board.State, i int) error {
		r := &b.Schedules[i]
		if r.Agent != seen {
			return errors.New("予定の担当が更新されました。最新の状態を確認してください")
		}
		r.Agent = agent
		return nil
	})
}

func Remove(s store.Store, id int) error {
	return editSchedule(s, id, func(b *board.State, i int) error {
		b.Schedules = append(b.Schedules[:i], b.Schedules[i+1:]...)
		return nil
	})
}

// 予定の位置は、他のペインでの解除を反映した最新のボードからロック内で求める。
func editSchedule(s store.Store, id int, change func(*board.State, int) error) error {
	return s.Edit(func(b *board.State) error {
		i := schedule.IndexByID(b.Schedules, id)
		if i < 0 {
			return fmt.Errorf("定期実行 #%d がありません", id)
		}
		return change(b, i)
	})
}
