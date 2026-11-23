package worker

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/store"
)

func TestWorkerRecordsCodexOutcome(t *testing.T) {
	for _, tc := range []struct {
		name, script, status, result string
	}{
		{"no final message", "exit 0", board.Done, "終了コード 0（最終メッセージなし）"},
		{"interrupted", "exit 130", board.Canceled, ""},
		{"failure with final message", "echo ' partial result ' > \"$3\"\nexit 7", board.Failed, "partial result\nexit status 7"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := store.Store{Dir: t.TempDir()}
			if err := s.Edit(func(b *board.State) error {
				b.NewCard("task", s.Dir, "")
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			if err := Start(s, "true", 1); err != nil {
				t.Fatal(err)
			}
			bin := t.TempDir()
			if err := os.WriteFile(filepath.Join(bin, "codex"), []byte("#!/bin/sh\n"+tc.script+"\n"), 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", bin+":"+os.Getenv("PATH"))
			t.Setenv("FACTORY_CARD", "1")
			t.Setenv("FACTORY_ATTEMPT", "1")
			if err := Run(s); err != nil {
				t.Fatal(err)
			}
			b, err := s.Read()
			if err != nil {
				t.Fatal(err)
			}
			c, err := b.FindCard(1)
			if err != nil {
				t.Fatal(err)
			}
			if c.Status != tc.status || c.Result != tc.result {
				t.Fatalf("status = %q, result = %q", c.Status, c.Result)
			}
		})
	}
}
