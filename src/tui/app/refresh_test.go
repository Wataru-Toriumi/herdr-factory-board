package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/store"
	"github.com/gdamore/tcell/v2"
)

func TestInputWhileHerdrListingIsPending(t *testing.T) {
	s := store.Store{Dir: t.TempDir()}
	bin := filepath.Join(s.Dir, "herdr")
	t.Setenv("LIST_GATE", filepath.Join(s.Dir, "release"))
	script := `#!/bin/sh
case "$1" in
	pane)
		touch "$LIST_GATE.started"
		while [ ! -f "$LIST_GATE" ]; do sleep 0.01; done
		echo '{"result":{"panes":[{"pane_id":"p1","agent":"codex"}]}}'
		;;
	workspace) echo '{"result":{"workspaces":[]}}' ;;
esac
`
	if err := os.WriteFile(bin, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	u := newBoardUI(s, bin)
	u.app.SetScreen(tcell.NewSimulationScreen("UTF-8"))
	tabChanged, listed := make(chan struct{}, 1), make(chan struct{}, 1)
	u.app.SetAfterDrawFunc(func(tcell.Screen) {
		if u.tab == agentTab {
			select {
			case tabChanged <- struct{}{}:
			default:
			}
		}
		if u.agents.GetItemCount() == 1 {
			select {
			case listed <- struct{}{}:
			default:
			}
		}
	})
	done := make(chan error, 1)
	go func() { done <- u.run() }()
	defer func() {
		_ = os.WriteFile(os.Getenv("LIST_GATE"), nil, 0600)
		u.app.QueueEvent(tcell.NewEventKey(tcell.KeyRune, 'q', 0))
		select {
		case err := <-done:
			if err != nil {
				t.Error(err)
			}
		case <-time.After(3 * time.Second):
			t.Error("画面が終了しませんでした")
		}
	}()
	deadline := time.Now().Add(2 * time.Second)
	for {
		if _, err := os.Stat(os.Getenv("LIST_GATE") + ".started"); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("ペインの取得が始まりませんでした")
		}
		time.Sleep(5 * time.Millisecond)
	}
	u.app.QueueEvent(tcell.NewEventKey(tcell.KeyTab, 0, 0))
	select {
	case <-tabChanged:
	case <-time.After(time.Second):
		t.Fatal("Herdr の応答待ちにタブを切り替えられませんでした")
	}
	if err := os.WriteFile(os.Getenv("LIST_GATE"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	select {
	case <-listed:
	case <-time.After(2 * time.Second):
		t.Fatal("取得したエージェントが画面に反映されませんでした")
	}
}

func TestBackgroundRefreshPreservesNewerLocalEdit(t *testing.T) {
	s := store.Store{Dir: t.TempDir()}
	u := newBoardUI(s, "true")
	old := readBoard(s)
	if err := s.Edit(func(b *board.State) error { b.NewCard("追加したカード", s.Dir, ""); return nil }); err != nil {
		t.Fatal(err)
	}
	u.refresh()
	u.applyRefresh(old, sessionSnapshot{})
	if c, ok := u.tasks.SelectedCard(); !ok || c.Prompt != "追加したカード" {
		t.Fatalf("更新後のカード: %+v, %t", c, ok)
	}
}
