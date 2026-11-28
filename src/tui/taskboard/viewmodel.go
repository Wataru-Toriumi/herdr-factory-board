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

func (mode Mode) Build(cards []board.Card, projects []string, workDir string, currentLane, selectedID int) ViewModel {
	model := ViewModel{SelectedLane: currentLane, SelectedRow: -1}
	var names map[string]string
	if workDir == "" {
		names = projectNames(projects)
	}
	for i := range cards {
		// 履歴は追加順を逆にたどり、各列の先頭に新しい実行を置く。
		index := i
		if mode == RunHistory {
			index = len(cards) - 1 - i
		}
		card := cards[index]
		if workDir != "" && card.Dir != workDir {
			continue
		}
		lane := mode.laneFor(card)
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
	} else if c.ScheduleID > 0 {
		label += fmt.Sprintf("  定期 #%d", c.ScheduleID)
	}
	if c.Status == board.Todo && c.SourceScheduleID() > 0 {
		label += "  起動待ち"
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
