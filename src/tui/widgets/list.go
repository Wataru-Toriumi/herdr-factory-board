// Package widgets は TUI の画面間で共有する端末部品を提供する。
package widgets

import (
	"reflect"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func NewList() *tview.List {
	l := tview.NewList().SetSelectedFocusOnly(true).SetHighlightFullLine(true).SetWrapAround(false)
	l.SetSecondaryTextColor(tcell.ColorLightSlateGray)
	l.SetSelectedStyle(tcell.StyleDefault.Background(tcell.ColorDarkCyan).Foreground(tcell.ColorWhite))
	l.SetBorder(true)
	return l
}

type RowList[T any] struct {
	*tview.List
	rows []T
}

func NewRowList[T any]() *RowList[T] {
	return &RowList[T]{List: NewList()}
}

func (l *RowList[T]) Selected() (T, bool) {
	i := l.GetCurrentItem()
	if i < 0 || i >= len(l.rows) {
		var zero T
		return zero, false
	}
	return l.rows[i], true
}

// 更新後も同じ項目を選ぶ。項目が消えたら、元の行番号に最も近い行を選ぶ。
func (l *RowList[T]) SetRows(rows []T, sameItem func(T, T) bool, labels func(T) (string, string)) {
	if reflect.DeepEqual(l.rows, rows) {
		return
	}
	selected, _ := l.Selected()
	index := l.GetCurrentItem()
	for i, row := range rows {
		if sameItem(row, selected) {
			index = i
		}
	}
	l.ReplaceRows(rows, index, labels)
}

func (l *RowList[T]) ReplaceRows(rows []T, index int, labels func(T) (string, string)) {
	l.Clear()
	for _, row := range rows {
		title, subtitle := labels(row)
		if first, _, multiline := strings.Cut(title, "\n"); multiline {
			title = strings.TrimSuffix(first, "\r") + " …"
		}
		l.AddItem(tview.Escape(title), tview.Escape(subtitle), 0, nil)
	}
	if len(rows) > 0 {
		l.SetCurrentItem(min(max(index, 0), len(rows)-1))
	}
	l.rows = rows
}

func (l *RowList[T]) Rows() []T { return l.rows }
