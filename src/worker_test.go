package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWorkerRecordsCodexOutcome(t *testing.T) {
	for _, tc := range []struct {
		name, script, status, result string
	}{
		{"no final message", "exit 0", done, "終了コード 0（最終メッセージなし）"},
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
