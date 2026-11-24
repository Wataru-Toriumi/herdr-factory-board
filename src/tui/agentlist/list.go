package agentlist

import (
	"github.com/Wataru-Toriumi/herdr-factory-board/src/tui/widgets"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const ListHint = "Tab 切替   ↑↓ 選択   Enter ペインへ   ? ヘルプ   q 閉じる"

type List struct {
	*widgets.RowList[Row]
	Blocked int
	focus   func(Row)
}

func NewList(focus func(Row)) *List {
	l := &List{RowList: widgets.NewRowList[Row](), focus: focus}
	l.SetTitle(" エージェント ").SetTitleAlign(tview.AlignLeft)
	return l
}

func (l *List) Render(model ViewModel) {
	l.Blocked = model.Blocked
	l.SetRows(model.Rows, func(a, b Row) bool { return a.Pane == b.Pane }, Row.Labels)
	l.SetTitle(model.Title())
}

func (l *List) HandleKey(event *tcell.EventKey) bool {
	if event.Key() != tcell.KeyEnter {
		return false
	}
	if row, ok := l.Selected(); ok {
		l.focus(row)
	}
	return true
}
