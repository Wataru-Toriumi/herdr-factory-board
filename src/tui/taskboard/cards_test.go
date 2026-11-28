package taskboard

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestLongCardTitleFitsLane(t *testing.T) {
	for _, width := range []int{39, 40, 140} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			screen := tcell.NewSimulationScreen("UTF-8")
			if err := screen.Init(); err != nil {
				t.Fatal(err)
			}
			defer screen.Fini()
			screen.SetSize(width, 6)
			v := NewView(Manual, nil, func(tview.Primitive) {})
			v.Resize(width)
			v.SetRect(0, 0, width, 6)
			v.Render([]board.Card{{ID: 1, Status: board.Todo, Prompt: strings.Repeat("あ", width)}}, nil, "")
			v.Draw(screen)
			x, y, w, _ := v.List(Todo).GetInnerRect()
			last, _, _, _ := screen.GetContent(x+w-1, y)
			gutter, _, _, _ := screen.GetContent(x+w, y)
			if last != '…' || gutter != ' ' {
				t.Fatalf("カード末尾と余白の文字が違います: %q %q", last, gutter)
			}
		})
	}
}

func TestCardHighlightFollowsSelectionAndScroll(t *testing.T) {
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	defer screen.Fini()
	screen.SetSize(40, 6)
	v := NewView(Manual, nil, func(tview.Primitive) {})
	v.Resize(40)
	v.SetRect(0, 0, 40, 6)
	v.Render([]board.Card{
		{ID: 1, Status: board.Todo, Prompt: "検索[red]"},
		{ID: 2, Status: board.Todo, Prompt: "表示を調整"},
		{ID: 3, Status: board.Todo, Prompt: "通知を追加"},
	}, nil, "")
	list := v.List(Todo)
	list.Focus(nil)
	v.Draw(screen)
	x, y, _, _ := list.GetInnerRect()
	check := func(y int, title rune, selected bool) {
		t.Helper()
		for line := 0; line < 2; line++ {
			mark, _, _, _ := screen.GetContent(x-2, y+line)
			r, _, style, _ := screen.GetContent(x, y+line)
			_, bg, attr := style.Decompose()
			if (mark == '▌') != selected || (bg == selectedSurface) != selected {
				t.Fatalf("選択表示が違います: 行=%d, 印=%c, 背景=%v", line, mark, bg)
			}
			if (attr&tcell.AttrBold != 0) != (line == 0) || (line == 0 && r != title) || (line == 1 && r != '#') {
				t.Fatalf("カードの文字または強調が違います: 行=%d, 文字=%c, 属性=%v", line, r, attr)
			}
		}
	}
	check(y, '検', true)
	check(y+2, '表', false)
	list.SetCurrentItem(2)
	v.Draw(screen)
	check(y, '表', false)
	check(y+2, '通', true)
	list.Blur()
	v.Draw(screen)
	check(y+2, '通', false)
}
