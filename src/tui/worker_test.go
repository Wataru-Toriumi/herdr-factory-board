package tui

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/application/worker"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/store"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/tui/taskboard"
	"github.com/gdamore/tcell/v2"
)

func TestCancelRunningWorker(t *testing.T) {
	s := store.Store{Dir: t.TempDir()}
	if err := s.Edit(func(b *board.State) error {
		b.NewCard("タスク", s.Dir, "").Status = board.Starting
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\ntouch ready\nsleep 30\n"
	if err := os.WriteFile(filepath.Join(s.Dir, "codex"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", s.Dir+":"+os.Getenv("PATH"))
	t.Setenv("FACTORY_CARD", "1")
	t.Setenv("FACTORY_ATTEMPT", "0")
	finished := make(chan error, 1)
	go func() { finished <- worker.Run(s) }()
	ctx, stop := context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()
	for {
		if _, err := os.Stat(filepath.Join(s.Dir, "ready")); err == nil {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("Codex が起動しませんでした")
		case err := <-finished:
			t.Fatalf("キャンセル前にワーカーが終了: %v", err)
		case <-time.After(10 * time.Millisecond):
		}
	}
	u := newBoardUI(s, "true")
	u.refresh()
	u.execute("result 1")
	u.key(tcell.NewEventKey(tcell.KeyRune, 'x', 0))
	select {
	case err := <-finished:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal("キャンセル後もワーカーが実行中です")
	}
	b, err := s.Read()
	if err != nil || taskboard.LaneFor(b.Cards[0]) != taskboard.FailedOrCanceled || u.detail.Card().Status != board.Canceled {
		t.Fatalf("キャンセル後のカード: %+v, %v", b, err)
	}
	u.key(tcell.NewEventKey(tcell.KeyRune, 'u', 0))
	if c := u.detail.Card(); c.Status != board.Todo || u.tasks.Lane() != taskboard.Todo || c.Dir != s.Dir || c.ID != 1 || c.Pane != "" {
		t.Fatalf("再オープン後のカード: %+v", c)
	}
	u.key(tcell.NewEventKey(tcell.KeyRune, 'd', 0))
	if c := u.detail.Card(); c.Status != board.Starting || c.Attempt != 1 || c.Phase != board.DesignPhase {
		t.Fatalf("再オープン後に設計を開始できません: %+v", c)
	}
}

func TestReopenAvailability(t *testing.T) {
	s := store.Store{Dir: t.TempDir()}
	for _, card := range []board.Card{
		{Status: board.Done},
		{Status: board.Canceled, Source: "定期 #3"},
		{Status: board.Canceled, ScheduleID: 3},
		{Status: board.Running, Phase: board.ImplementationPhase},
	} {
		card.ID = 1
		if err := s.Edit(func(b *board.State) error { b.Cards = []board.Card{card}; return nil }); err != nil {
			t.Fatal(err)
		}
		u := newBoardUI(s, "true")
		u.execute("result 1")
		event := tcell.NewEventKey(tcell.KeyRune, 'u', 0)
		if u.key(event) != event || worker.Reopen(s, card) == nil {
			t.Fatalf("対象外のカードを再オープンできました: %+v", card)
		}
	}
}
