package main

import (
	"fmt"
	"reflect"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// 自動更新時は、選択中のカード番号を使って移動先の列と行を復元する。
func (u *boardUI) renderTasks(cards []card) {
	groups := make([][]card, len(laneNames))
	for _, c := range cards {
		if i := cardLane(c.Status); i >= 0 {
			groups[i] = append(groups[i], c)
		}
	}
	selected, _ := u.selectedCard()
	for i, group := range groups {
		list := u.lists[i]
		if !reflect.DeepEqual(u.cards[i], group) {
			index := list.GetCurrentItem()
			list.Clear()
			for j, c := range group {
				label := fmt.Sprintf("#%d", c.ID)
				if c.Source != "" {
					label += "  " + c.Source
				}
				if c.Status == starting {
					label += "  起動中"
				}
				list.AddItem(tview.Escape(c.Prompt), tview.Escape(label), 0, nil)
				if selected.ID == c.ID {
					index = j
					if u.lane != i {
						u.lane = i
						u.layoutColumns()
						if u.activePage() == "tasks" {
							u.app.SetFocus(list)
						}
					}
				}
			}
			if len(group) > 0 {
				list.SetCurrentItem(min(max(0, index), len(group)-1))
			}
			u.cards[i] = group
		}
		list.SetTitle(fmt.Sprintf(" %s %d ", laneNames[i], len(group)))
	}
	u.highlightLane()
}

func (u *boardUI) highlightLane() {
	for i, list := range u.lists {
		border := tcell.ColorDarkSlateGray
		if i == u.lane {
			border = laneColors[i]
		}
		list.SetBorderColor(border)
	}
}

var agentStates = map[string]string{
	"blocked": "入力待ち", "working": "作業中", "idle": "待機中", "done": "完了",
}

func (u *boardUI) renderAgents(rows []agentRow) {
	if !reflect.DeepEqual(rows, u.rows) {
		index := u.agents.GetCurrentItem()
		var id string
		if index >= 0 && index < len(u.rows) {
			id = u.rows[index].pane
		}
		u.agents.Clear()
		for i, r := range rows {
			if r.pane == id {
				index = i
			}
			state := agentStates[r.status]
			if state == "" {
				state = "状態不明"
			}
			u.agents.AddItem(tview.Escape(fmt.Sprintf("%s  ·  %s   %s", state, r.name, r.task)), tview.Escape(r.workspace+"   "+r.pane), 0, nil)
		}
		if len(rows) > 0 {
			u.agents.SetCurrentItem(min(max(index, 0), len(rows)-1))
		}
		u.rows = rows
	}
	u.agents.SetTitle(fmt.Sprintf(" エージェント %d ", len(rows)))
	if len(rows) == 0 {
		u.agents.SetTitle(" エージェントなし ")
	}
	u.updateHeader()
}
