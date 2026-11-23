package tui

import (
	"fmt"
	"maps"
	"reflect"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/schedule"
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
	for i, group := range groups {
		list := u.lists[i]
		if labelsChanged || !reflect.DeepEqual(u.cards[i], group) {
			index := list.GetCurrentItem()
			list.Clear()
			for j, c := range group {
				list.AddItem(tview.Escape(c.Prompt), tview.Escape(cardListLabel(c, i, names[c.Dir])), 0, nil)
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

type scheduleList struct {
	*tview.List
	all   []schedule.Rule
	rules []schedule.Rule
}

func (l *scheduleList) byID(id int) (schedule.Rule, bool) {
	for _, r := range l.all {
		if r.ID == id {
			return r, true
		}
	}
	return schedule.Rule{}, false
}

func (l *scheduleList) selected() (schedule.Rule, bool) {
	i := l.GetCurrentItem()
	if i < 0 || i >= len(l.rules) {
		return schedule.Rule{}, false
	}
	return l.rules[i], true
}

func (l *scheduleList) render(all []schedule.Rule, dir string, active bool, err error) {
	l.all = all
	rules := all
	if dir != "" {
		rules = nil
		for _, r := range all {
			if r.Dir == dir {
				rules = append(rules, r)
			}
		}
	}
	state := "停止中"
	if err != nil {
		state = "状態不明"
	} else if active {
		state = "稼働中"
	}
	l.SetTitle(fmt.Sprintf(" 定期実行 %d件 · %s ", len(rules), state))
	if reflect.DeepEqual(l.rules, rules) {
		return
	}

	selected, _ := l.selected()
	index := l.GetCurrentItem()
	l.Clear()
	for i, r := range rules {
		l.AddItem(tview.Escape(fmt.Sprintf("%s  %s", r.Frequency(), r.Prompt)), tview.Escape(fmt.Sprintf("#%d  %s", r.ID, r.Dir)), 0, nil)
		if r.ID == selected.ID {
			index = i
		}
	}
	if len(rules) > 0 {
		l.SetCurrentItem(min(max(0, index), len(rules)-1))
	}
	l.rules = rules
}
