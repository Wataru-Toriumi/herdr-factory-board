package tui

import (
	"fmt"
	"maps"
	"slices"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/gdamore/tcell/v2"
)

func cardListLabel(c board.Card, lane int, project string) string {
	label := fmt.Sprintf("#%d", c.ID)
	if project != "" {
		label += "  " + project
	}
	if c.Source != "" {
		label += "  " + c.Source
	}
	if c.Status == board.Starting {
		label += "  起動中"
	}
	phase := c.PhaseLabel()
	if phase == "" {
		return label
	}
	if lane == failedOrCanceledLane {
		return phase + "  " + label
	}
	return label + "  " + phase
}

// 自動更新時は、選択中のカード番号を使って移動先の列と行を復元する。
func (u *boardUI) renderTasks(cards []board.Card) {
	var names map[string]string
	if u.workDir == "" {
		names = projectNames(u.projects)
	}
	labelsChanged := !maps.Equal(names, u.taskProjectNames)
	u.taskProjectNames = names
	var groups [len(taskLanes)][]board.Card
	for _, c := range cards {
		if u.workDir != "" && c.Dir != u.workDir {
			continue
		}
		if i := cardLane(c); i >= 0 {
			groups[i] = append(groups[i], c)
		}
	}
	selected, _ := u.selectedCard()
	isSelected := func(c board.Card) bool { return c.ID == selected.ID }
	selectedLane := u.lane
	for i, group := range groups {
		list := u.lists[i]
		if labelsChanged || !slices.Equal(list.rows, group) {
			found := list.replaceRows(group, isSelected, func(c board.Card) (string, string) {
				return c.Prompt, cardListLabel(c, i, names[c.Dir])
			})
			if found {
				selectedLane = i
			}
		}
		list.SetTitle(fmt.Sprintf(" %s %d ", taskLanes[i].name, len(group)))
	}
	u.selectLane(selectedLane)
}

func (u *boardUI) selectLane(lane int) {
	if u.lane != lane {
		u.lane = lane
		u.layoutColumns()
		// 入力欄や詳細画面を操作中でも、戻ったときに表示する列は更新する。
		if u.activePage() == u.tabs[taskTab].page {
			u.app.SetFocus(u.lists[lane])
		}
	}
	for i, list := range u.lists {
		border := tcell.ColorDarkSlateGray
		if i == u.lane {
			border = taskLanes[i].color
		}
		list.SetBorderColor(border)
	}
}
