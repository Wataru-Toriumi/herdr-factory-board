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
	_, err := RegisterRule(s, herdrBin, schedule.Rule{Agent: agent, Prompt: prompt, Dir: dir}, kind, when)
	return err
}

// RegisterRule は保存した予定番号を返す。保存後の起動失敗でも同じ予定を二重登録しないために使う。
func RegisterRule(s store.Store, herdrBin string, rule schedule.Rule, kind, when string) (int, error) {
	rule.Prompt = strings.TrimSpace(rule.Prompt)
	if rule.Prompt == "" {
		return 0, errors.New("作業内容を入力してください")
	}
	var id int
	if err := s.Edit(func(b *board.State) error {
		if rule.Dir == "" {
			return errors.New("先に dir で作業先を指定してください")
		}
		if err := rule.SetTiming(kind, when, time.Now()); err != nil {
			return err
		}
		id = b.AddSchedule(rule)
		return nil
	}); err != nil {
		return 0, err
	}
	if err := EnsureRunning(s, herdrBin); err != nil {
		return id, fmt.Errorf("予定 #%d は保存済みです。scheduler で起動を再試行してください（%w）", id, err)
	}
	return id, nil
}

func UpdateTiming(s store.Store, id int, value string) error {
	return updateTiming(s, id, value, nil)
}

// TimingConflict は編集開始後に他のペインが保存した設定を返す。
type TimingConflict struct {
	Current schedule.Rule
}

func (e *TimingConflict) Error() string {
	return "別のペインで予定が変更されました。現在の設定は " + e.Current.Frequency() + " です。"
}

func ChangeTiming(s store.Store, seen schedule.Rule, value string) error {
	return updateTiming(s, seen.ID, value, &seen)
}

func updateTiming(s store.Store, id int, value string, seen *schedule.Rule) error {
	return editSchedule(s, id, func(b *board.State, i int) error {
		r := &b.Schedules[i]
		if seen != nil && (r.TimingKind() != seen.TimingKind() || r.TimingValue() != seen.TimingValue()) {
			return &TimingConflict{Current: *r}
		}
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

// PromptConflict は編集中に別のペインで保存された実行内容を返す。
type PromptConflict struct {
	Current schedule.Rule
}

func (e *PromptConflict) Error() string {
	return "別のペインで実行内容が変更されました。最新の内容を確認してください"
}

func ChangePrompt(s store.Store, seen schedule.Rule, value string) error {
	return editSchedule(s, seen.ID, func(b *board.State, i int) error {
		r := &b.Schedules[i]
		if r.Prompt != seen.Prompt {
			return &PromptConflict{Current: *r}
		}
		return setPrompt(r, value)
	})
}

func setPrompt(r *schedule.Rule, value string) error {
	prompt := strings.TrimSpace(value)
	if prompt == "" {
		return errors.New("実行内容を入力してください")
	}
	r.Prompt = prompt
	return nil
}

// ChangeSettings は編集中の設定をまとめて保存し、競合時は最新の予定を返す。
func ChangeSettings(s store.Store, seen schedule.Rule, timing, prompt, agent string) (*schedule.Rule, error) {
	var conflict *schedule.Rule
	err := editSchedule(s, seen.ID, func(b *board.State, i int) error {
		r := &b.Schedules[i]
		if r.TimingKind() != seen.TimingKind() || r.TimingValue() != seen.TimingValue() || r.Prompt != seen.Prompt || r.Agent != seen.Agent {
			conflict = r
			return errors.New("別のペインで予定が変更されました。最新の設定を確認してください")
		}
		if err := setPrompt(r, prompt); err != nil {
			return err
		}
		if timing != r.TimingValue() {
			if err := r.SetTiming(r.TimingKind(), timing, time.Now()); err != nil {
				return err
			}
		}
		r.Agent = agent
		return nil
	})
	return conflict, err
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
