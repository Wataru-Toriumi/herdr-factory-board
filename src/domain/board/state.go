// Package board はボードに保存するカード、予定、プロジェクトの状態を扱う。
package board

import (
	"fmt"
	"slices"
	"time"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/schedule"
)

type State struct {
	NextID    int
	Dir       string
	Cards     []Card
	Schedules []schedule.Rule
	Projects  []string `json:",omitempty"`
}

func (b *State) allocateID() int {
	b.NextID++
	return b.NextID
}

func (b *State) NewCard(prompt, dir, source string) *Card {
	b.Cards = append(b.Cards, Card{ID: b.allocateID(), Prompt: prompt, Dir: dir, Source: source, Status: Todo})
	return &b.Cards[len(b.Cards)-1]
}

// AddSchedule はカードと共通の通し番号を付けて予定を登録する。
func (b *State) AddSchedule(rule schedule.Rule) int {
	rule.ID = b.allocateID()
	b.Schedules = append(b.Schedules, rule)
	return rule.ID
}

func (b State) HasDueSchedule(now time.Time) bool {
	for _, r := range b.Schedules {
		if r.Due(now) {
			return true
		}
	}
	return false
}

// RecordDueSchedules は期限を迎えた各予定の実行を記録し、対応する履歴カードを作る。
func (b *State) RecordDueSchedules(now time.Time) {
	for i := range b.Schedules {
		r := &b.Schedules[i]
		if r.Due(now) {
			source := r.RecordOccurrence(now)
			card := b.NewCard(r.Prompt, r.Dir, source)
			card.ScheduleID, card.Agent = r.ID, r.Agent
		}
	}
}

func (b *State) FindCard(id int) (*Card, error) {
	for i := range b.Cards {
		if b.Cards[i].ID == id {
			return &b.Cards[i], nil
		}
	}
	return nil, fmt.Errorf("カード #%d がありません", id)
}

// 旧版の作業先と履歴もプロジェクトとして扱う。カードや予定の書き換えは不要。
func (b State) ProjectDirs() []string {
	seen := make(map[string]bool)
	add := func(dir string) {
		if dir != "" {
			seen[dir] = true
		}
	}
	add(b.Dir)
	for _, dir := range b.Projects {
		add(dir)
	}
	for _, c := range b.Cards {
		add(c.Dir)
	}
	for _, r := range b.Schedules {
		add(r.Dir)
	}
	dirs := make([]string, 0, len(seen))
	for dir := range seen {
		dirs = append(dirs, dir)
	}
	slices.Sort(dirs)
	return dirs
}
