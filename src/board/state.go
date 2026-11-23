// Package board はボードに保存するカード、予定、プロジェクトの状態を扱う。
package board

import (
	"fmt"
	"slices"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/schedule"
)

type State struct {
	NextID    int
	Dir       string
	Cards     []Card
	Schedules []schedule.Rule
	Projects  []string `json:",omitempty"`
}

func (b *State) NewCard(prompt, dir, source string) *Card {
	b.NextID++
	b.Cards = append(b.Cards, Card{ID: b.NextID, Prompt: prompt, Dir: dir, Source: source, Status: Todo})
	return &b.Cards[len(b.Cards)-1]
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
