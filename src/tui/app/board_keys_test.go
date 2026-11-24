package app

import (
	"testing"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/store"
	"github.com/gdamore/tcell/v2"
)

func TestListKeyFallback(t *testing.T) {
	for _, tc := range []struct {
		page        string
		tab         int
		commandKey  rune
		reservedKey rune
	}{
		{"tasks", taskTab, 'i', 'v'},
		{"agents", agentTab, 'v', 0},
		{"automation", automationTab, 'v', 'x'},
	} {
		t.Run(tc.page, func(t *testing.T) {
			u := newBoardUI(store.Store{Dir: t.TempDir()}, "")
			u.tab = tc.tab
			u.switchView()
			down := tcell.NewEventKey(tcell.KeyDown, 0, 0)
			if u.key(down) != down {
				t.Fatal("上下移動のキーが一覧に届きません")
			}
			if tc.tab != taskTab && u.key(tcell.NewEventKey(tcell.KeyRight, 0, 0)) != nil {
				t.Fatal("レーン移動のキーが別の一覧に渡されました")
			}
			if tc.reservedKey != 0 {
				if u.key(tcell.NewEventKey(tcell.KeyRune, tc.reservedKey, 0)) != nil || u.activePage() != tc.page {
					t.Fatal("選択行がないときに専用キーが一覧以外で処理されました")
				}
			}
			u.key(tcell.NewEventKey(tcell.KeyRune, tc.commandKey, 0))
			if u.activePage() != "input" || u.input.GetText() != string(tc.commandKey) {
				t.Fatal("タブに割り当てられていない文字でコマンド入力を始められません")
			}
		})
	}
}
