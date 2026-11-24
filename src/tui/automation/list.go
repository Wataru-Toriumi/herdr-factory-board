package automation

import (
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/schedule"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/tui/widgets"
	"github.com/gdamore/tcell/v2"
)

const ListHint = "n 毎日  i 間隔  t 変更  p プロジェクト  ↑↓ 選択  Enter 詳細  x 解除  Tab 切替  ? ヘルプ"

type ListActions struct {
	New        func(kind string)
	Open       func(rule schedule.Rule)
	EditTiming func(rule schedule.Rule)
	Remove     func(rule schedule.Rule)
}

type List struct {
	*widgets.RowList[schedule.Rule]
	actions ListActions
	model   ViewModel
}

func NewList(actions ListActions) *List {
	return &List{RowList: widgets.NewRowList[schedule.Rule](), actions: actions}
}

func (l *List) ByID(id int) (schedule.Rule, bool) {
	return l.model.ByID(id)
}

func (l *List) Render(rules []schedule.Rule, dir string, active bool, err error) {
	l.model = Build(rules, dir, active, err)
	l.SetTitle(l.model.Title)
	l.SetRows(l.model.Rules, func(a, b schedule.Rule) bool { return a.ID == b.ID }, RowLabels)
}

func (l *List) HandleKey(event *tcell.EventKey) bool {
	switch event.Key() {
	case tcell.KeyEnter:
		if r, ok := l.Selected(); ok {
			l.actions.Open(r)
		}
	case tcell.KeyRune:
		switch event.Rune() {
		case 'n':
			l.actions.New(schedule.Daily)
		case 'i':
			l.actions.New(schedule.Every)
		case 't':
			if r, ok := l.Selected(); ok {
				l.actions.EditTiming(r)
			}
		case 'x':
			if r, ok := l.Selected(); ok {
				l.actions.Remove(r)
			}
		default:
			return false
		}
	default:
		return false
	}
	return true
}
