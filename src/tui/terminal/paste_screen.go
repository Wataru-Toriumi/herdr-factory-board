// Package terminal は TUI の端末入力を扱う。
package terminal

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// tview v0.42.0 が貼り付け中の LF を読み飛ばすため、CR・LF・CRLF を Enter に揃える。
type pasteScreen struct {
	tcell.Screen
	pasting, afterCR bool
	initErr          error
}

// SetScreen は貼り付け時の改行を正規化するスクリーンを初期化する。
func SetScreen(app *tview.Application, raw tcell.Screen) error {
	screen := &pasteScreen{Screen: raw}
	app.SetScreen(screen)
	return screen.initErr
}

// SetScreen は初期化エラーを返さないため、呼び出し元で確認できるようにする。
func (s *pasteScreen) Init() error {
	s.initErr = s.Screen.Init()
	return s.initErr
}

func (s *pasteScreen) PollEvent() tcell.Event {
	for {
		event := s.Screen.PollEvent()
		switch e := event.(type) {
		case *tcell.EventPaste:
			s.pasting, s.afterCR = e.Start(), false
		case *tcell.EventKey:
			if s.pasting {
				afterCR := s.afterCR
				s.afterCR = e.Key() == tcell.KeyEnter
				if e.Key() == tcell.KeyCtrlJ {
					if afterCR {
						continue
					}
					return tcell.NewEventKey(tcell.KeyEnter, '\r', tcell.ModNone)
				}
			}
		}
		return event
	}
}
