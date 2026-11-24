package taskboard

import (
	"fmt"
	"slices"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/tui/widgets"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const ListHint = "v 詳細  n 追加  p プロジェクト  ? ヘルプ  Tab 切替  ←→ 列  ↑↓ 選択  Enter 操作  q 閉じる"

type View struct {
	*tview.Flex
	lists     [LaneCount]*widgets.RowList[Row]
	lane      int
	compact   bool
	execute   func(string)
	focusLane func(tview.Primitive)
}

func NewView(execute func(string), focusLane func(tview.Primitive)) *View {
	v := &View{Flex: tview.NewFlex(), execute: execute, focusLane: focusLane}
	for i, lane := range taskLanes {
		v.lists[i] = widgets.NewRowList[Row]()
		v.lists[i].SetTitleColor(lane.color)
		v.lists[i].SetBorderPadding(0, 0, 3, 1)
	}
	v.layoutColumns()
	return v
}

func (v *View) Resize(width int) {
	if narrow := width < 17*LaneCount; narrow != v.compact {
		v.compact = narrow
		v.layoutColumns()
	}
}

func (v *View) Lane() int                           { return v.lane }
func (v *View) List(lane int) *widgets.RowList[Row] { return v.lists[lane] }

func (v *View) layoutColumns() {
	v.Flex.Clear()
	for i, l := range v.lists {
		if !v.compact || i == v.lane {
			v.Flex.AddItem(l, 0, 1, false)
		}
	}
}

func (v *View) SelectedCard() (board.Card, bool) {
	row, ok := v.lists[v.lane].Selected()
	return row.Card, ok
}

func (v *View) Render(cards []board.Card, projects []string, dir string) {
	selected, _ := v.SelectedCard()
	model := Build(cards, projects, dir, v.lane, selected.ID)
	for lane, rows := range model.Rows {
		list := v.lists[lane]
		if !slices.Equal(list.Rows(), rows) {
			index := list.GetCurrentItem()
			if lane == model.SelectedLane && model.SelectedRow >= 0 {
				index = model.SelectedRow
			}
			list.ReplaceRows(rows, index, Row.Labels)
		}
		list.SetTitle(fmt.Sprintf(" %s %d ", taskLanes[lane].name, len(rows)))
	}
	v.selectLane(model.SelectedLane)
}

func (v *View) selectLane(lane int) {
	if v.lane != lane {
		v.lane = lane
		v.layoutColumns()
		// 入力欄や詳細画面を操作中でも、戻ったときに表示する列は更新する。
		v.focusLane(v.lists[lane])
	}
	for i, list := range v.lists {
		border := tcell.ColorDarkSlateGray
		if i == v.lane {
			border = taskLanes[i].color
		}
		list.SetBorderColor(border)
	}
}

func (v *View) HandleKey(event *tcell.EventKey) bool {
	switch event.Key() {
	case tcell.KeyLeft:
		v.selectLane((v.lane + len(taskLanes) - 1) % len(taskLanes))
	case tcell.KeyRight:
		v.selectLane((v.lane + 1) % len(taskLanes))
	case tcell.KeyEnter:
		v.activateCard()
	case tcell.KeyRune:
		if event.Rune() != 'v' {
			return false
		}
		if c, ok := v.SelectedCard(); ok {
			v.execute(fmt.Sprintf("result %d", c.ID))
		}
	default:
		return false
	}
	return true
}

func (v *View) activateCard() {
	c, ok := v.SelectedCard()
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
	v.execute(fmt.Sprintf("%s %d", action, c.ID))
}
