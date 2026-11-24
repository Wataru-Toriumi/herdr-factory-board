package tui

import (
	"fmt"
	"slices"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/tui/taskboard"
	"github.com/gdamore/tcell/v2"
)

func (u *boardUI) renderTasks(cards []board.Card) {
	selected, _ := u.selectedCard()
	model := taskboard.Build(cards, u.projects, u.workDir, u.lane, selected.ID)
	for lane, rows := range model.Rows {
		list := u.lists[lane]
		if !slices.Equal(list.rows, rows) {
			index := list.GetCurrentItem()
			if lane == model.SelectedLane && model.SelectedRow >= 0 {
				index = model.SelectedRow
			}
			list.replaceRows(rows, index, taskboard.Row.Labels)
		}
		list.SetTitle(fmt.Sprintf(" %s %d ", taskLanes[lane].name, len(rows)))
	}
	u.selectLane(model.SelectedLane)
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

func (u *boardUI) taskListKey(event *tcell.EventKey) bool {
	switch event.Key() {
	case tcell.KeyLeft:
		u.selectLane((u.lane + len(taskLanes) - 1) % len(taskLanes))
	case tcell.KeyRight:
		u.selectLane((u.lane + 1) % len(taskLanes))
	case tcell.KeyEnter:
		u.activateCard()
	case tcell.KeyRune:
		if event.Rune() != 'v' {
			return false
		}
		if c, ok := u.selectedCard(); ok {
			u.execute(fmt.Sprintf("result %d", c.ID))
		}
	default:
		return false
	}
	return true
}

func (u *boardUI) activateCard() {
	c, ok := u.selectedCard()
	if !ok {
		return
	}
	action := "result"
	switch c.Status {
	case board.Todo:
		action = "start"
	case board.Running:
		action = "open"
	}
	u.execute(fmt.Sprintf("%s %d", action, c.ID))
}
