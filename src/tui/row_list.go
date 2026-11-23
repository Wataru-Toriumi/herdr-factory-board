package tui

import (
	"reflect"

	"github.com/rivo/tview"
)

type rowList[T any] struct {
	*tview.List
	rows []T
}

func newRowList[T any]() rowList[T] {
	return rowList[T]{List: boardList()}
}

func (l *rowList[T]) selected() (T, bool) {
	i := l.GetCurrentItem()
	if i < 0 || i >= len(l.rows) {
		var zero T
		return zero, false
	}
	return l.rows[i], true
}

// 更新後も同じ項目を選ぶ。項目が消えたら、元の行番号に最も近い行を選ぶ。
func (l *rowList[T]) setRows(rows []T, sameItem func(T, T) bool, labels func(T) (string, string)) {
	if reflect.DeepEqual(l.rows, rows) {
		return
	}
	index := l.GetCurrentItem()
	selected, _ := l.selected()
	l.Clear()
	for i, row := range rows {
		title, subtitle := labels(row)
		l.AddItem(tview.Escape(title), tview.Escape(subtitle), 0, nil)
		if sameItem(row, selected) {
			index = i
		}
	}
	if len(rows) > 0 {
		l.SetCurrentItem(min(max(index, 0), len(rows)-1))
	}
	l.rows = rows
}
