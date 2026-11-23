package tui

import (
	"fmt"
	"maps"
	"reflect"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
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
	groups := make([][]board.Card, len(laneNames))
	for _, c := range cards {
		if u.workDir != "" && c.Dir != u.workDir {
			continue
		}
		if i := cardLane(c); i >= 0 {
			groups[i] = append(groups[i], c)
		}
	}
	selected, _ := u.selectedCard()
	selectedLane := u.lane
	for i, group := range groups {
		list := u.lists[i]
		if labelsChanged || !reflect.DeepEqual(u.cards[i], group) {
			index := list.GetCurrentItem()
			list.Clear()
			for j, c := range group {
				list.AddItem(tview.Escape(c.Prompt), tview.Escape(cardListLabel(c, i, names[c.Dir])), 0, nil)
				if selected.ID == c.ID {
					index = j
					selectedLane = i
				}
			}
			if len(group) > 0 {
				list.SetCurrentItem(min(max(0, index), len(group)-1))
			}
			u.cards[i] = group
		}
		list.SetTitle(fmt.Sprintf(" %s %d ", laneNames[i], len(group)))
	}
	u.selectLane(selectedLane)
}

func (u *boardUI) selectLane(lane int) {
	if u.lane != lane {
		u.lane = lane
		u.layoutColumns()
		// 入力欄や詳細画面を操作中でも、戻ったときに表示する列は更新する。
		if u.activePage() == "tasks" {
			u.app.SetFocus(u.lists[lane])
		}
	}
	for i, list := range u.lists {
		border := tcell.ColorDarkSlateGray
		if i == u.lane {
			border = laneColors[i]
		}
		list.SetBorderColor(border)
	}
}
