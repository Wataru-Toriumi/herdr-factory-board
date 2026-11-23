package tui

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/store"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/worker"
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
	if err != nil || cardLane(b.Cards[0]) != failedOrCanceledLane || u.detailCard.Status != board.Canceled {
		t.Fatalf("キャンセル後のカード: %+v, %v", b, err)
	}
}
