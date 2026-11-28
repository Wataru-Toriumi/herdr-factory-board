package taskboard

import (
	"github.com/Wataru-Toriumi/herdr-factory-board/src/tui/widgets"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var (
	cardSurfaces    = [...]tcell.Color{tcell.NewHexColor(0x17212c), tcell.NewHexColor(0x1d2a37)}
	selectedSurface = tcell.NewHexColor(0x30475c)
	cardTitle       = tcell.NewHexColor(0xe8eff5)
	cardMetadata    = tcell.NewHexColor(0xb2c1cf)
)

func (v *View) Draw(screen tcell.Screen) {
	v.Flex.Draw(screen)
	for _, lane := range v.columns {
		if !v.compact || lane == v.lane {
			drawCards(screen, v.lists[lane], taskLanes[lane].color)
		}
	}
}

// 一覧が決定したスクロール位置と描画済みの文字を使い、表示中のカードだけを装飾する。
func drawCards(screen tcell.Screen, list *widgets.RowList[Row], accent tcell.Color) {
	x, y, width, height := list.GetInnerRect()
	if width == 0 {
		return
	}
	first, horizontal := list.GetOffset()
	first = max(0, first)
	for line := 0; line < height; line++ {
		index := first + line/2
		if index >= list.GetItemCount() {
			break
		}
		surface := cardSurfaces[index%len(cardSurfaces)]
		marker, ink := '╭', cardTitle
		label, metadata := list.GetItemText(index)
		if line%2 == 1 {
			marker, ink = '╰', cardMetadata
			label = metadata
		}
		if list.HasFocus() && index == list.GetCurrentItem() {
			surface, marker = selectedSurface, '▌'
		}
		style := tcell.StyleDefault.Background(surface)
		screen.SetContent(x-2, y+line, marker, nil, style.Foreground(accent))
		screen.SetContent(x-1, y+line, ' ', nil, style)
		textStyle := style.Foreground(ink).Bold(line%2 == 0)
		clipped := tview.TaggedStringWidth(label) > horizontal+width
		for col := x; col < x+width; {
			char, combining, _, cells := screen.GetContent(col, y+line)
			// 末尾が全角文字でも、省略記号までカードの幅に収める。
			if clipped && col+cells >= x+width {
				screen.SetContent(col, y+line, ' ', nil, textStyle)
				screen.SetContent(x+width-1, y+line, '…', nil, textStyle)
				break
			}
			screen.SetContent(col, y+line, char, combining, textStyle)
			col += max(1, cells)
		}
	}
}
