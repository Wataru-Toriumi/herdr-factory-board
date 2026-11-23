// Package taskboard はタスクボードの表示行と選択位置を組み立てる。
package taskboard

import (
	"fmt"
	"path/filepath"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
)

type Row struct {
	Card    board.Card
	project string
}

type ViewModel struct {
	Rows         [LaneCount][]Row
	SelectedLane int
	// 選択中のカードが表示対象から外れた場合は -1。画面は元の行番号を使う。
	SelectedRow int
}

func Build(cards []board.Card, projects []string, workDir string, currentLane, selectedID int) ViewModel {
	model := ViewModel{SelectedLane: currentLane, SelectedRow: -1}
	var names map[string]string
	if workDir == "" {
		names = projectNames(projects)
	}
	for _, card := range cards {
		if workDir != "" && card.Dir != workDir {
			continue
		}
		lane := LaneFor(card)
		if lane < 0 {
			continue
		}
		if card.ID == selectedID {
			model.SelectedLane, model.SelectedRow = lane, len(model.Rows[lane])
		}
		model.Rows[lane] = append(model.Rows[lane], Row{Card: card, project: names[card.Dir]})
	}
	return model
}

func (row Row) Labels() (string, string) {
	c := row.Card
	label := fmt.Sprintf("#%d", c.ID)
	if row.project != "" {
		label += "  " + row.project
	}
	if c.Source != "" {
		label += "  " + c.Source
	}
	if c.Status == board.Starting {
		label += "  起動中"
	}
	phase := c.PhaseLabel()
	if phase == "" {
		return c.Prompt, label
	}
	if LaneFor(c) == FailedOrCanceled {
		return c.Prompt, phase + "  " + label
	}
	return c.Prompt, label + "  " + phase
}

func projectNames(dirs []string) map[string]string {
	counts := make(map[string]int)
	for _, dir := range dirs {
		counts[filepath.Base(dir)]++
	}
	names := make(map[string]string, len(dirs))
	for _, dir := range dirs {
		name := filepath.Base(dir)
		if counts[name] > 1 {
			name = dir
		}
		names[dir] = name
	}
	return names
}
