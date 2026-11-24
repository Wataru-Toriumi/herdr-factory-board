// Package tui は Factory Board の端末画面を起動する。
package tui

import (
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/store"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/tui/app"
	"github.com/gdamore/tcell/v2"
)

func Run(s store.Store, herdr string) error {
	screen, err := tcell.NewScreen()
	if err != nil {
		return err
	}
	return app.Run(s, herdr, screen)
}
