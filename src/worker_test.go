package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
)

func TestWorkerRecordsCodexOutcome(t *testing.T) {
	for _, tc := range []struct {
		name, script, status, result string
	}{
		{"no final message", "exit 0", done, "終了コード 0（最終メッセージなし）"},
		{"interrupted", "exit 130", canceled, ""},
		{"failure with final message", "echo ' partial result ' > \"$3\"\nexit 7", failed, "partial result\nexit status 7"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := store{t.TempDir()}
			if err := s.edit(func(b *board) error {
				b.newCard("task", s.dir, "")
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			if err := startCard(s, "true", 1); err != nil {
				t.Fatal(err)
			}
			bin := t.TempDir()
			if err := os.WriteFile(filepath.Join(bin, "codex"), []byte("#!/bin/sh\n"+tc.script+"\n"), 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", bin+":"+os.Getenv("PATH"))
			t.Setenv("FACTORY_CARD", "1")
			t.Setenv("FACTORY_ATTEMPT", "1")
			if err := runWorker(s); err != nil {
				t.Fatal(err)
			}
			b, err := s.read()
			if err != nil {
				t.Fatal(err)
			}
			c, err := b.find(1)
			if err != nil {
				t.Fatal(err)
			}
			if c.Status != tc.status || c.Result != tc.result {
				t.Fatalf("status = %q, result = %q", c.Status, c.Result)
			}
		})
	}
}

func TestCancelRunningWorker(t *testing.T) {
	s := store{t.TempDir()}
	if err := s.edit(func(b *board) error {
		b.newCard("タスク", s.dir, "").Status = starting
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\ntouch ready\nsleep 30\n"
	if err := os.WriteFile(filepath.Join(s.dir, "codex"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", s.dir+":"+os.Getenv("PATH"))
	t.Setenv("FACTORY_CARD", "1")
	t.Setenv("FACTORY_ATTEMPT", "0")
	finished := make(chan error, 1)
	go func() { finished <- runWorker(s) }()
	ctx, stop := context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()
	for {
		if _, err := os.Stat(filepath.Join(s.dir, "ready")); err == nil {
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
	b, err := s.read()
	if err != nil || cardLane(b.Cards[0]) != failedOrCanceledLane || u.detailCard.Status != canceled {
		t.Fatalf("キャンセル後のカード: %+v, %v", b, err)
	}
}
