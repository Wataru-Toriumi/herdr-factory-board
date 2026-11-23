package worker

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/store"
)

func TestRecoverInterruptedCard(t *testing.T) {
	const invalidState = "中断した実行だけを復旧できます"
	for _, tc := range []struct {
		status  string
		locked  bool
		wantErr string
	}{
		{board.Running, true, "担当ペインは実行中です"},
		{board.Starting, false, ""},
		{board.Running, false, ""},
		{board.Todo, false, invalidState},
		{board.AwaitingDesignReview, false, invalidState},
		{board.AwaitingResultReview, false, invalidState},
		{board.Canceled, false, invalidState},
		{board.Failed, false, invalidState},
		{board.Done, false, invalidState},
	} {
		name := tc.status
		if tc.locked {
			name += "/ロック中"
		}
		t.Run(name, func(t *testing.T) {
			s := store.Store{Dir: t.TempDir()}
			card := board.Card{ID: 7, Status: tc.status, Attempt: 3, Phase: board.ImplementationPhase, Pane: "pane-1", Result: "前回の結果"}
			if err := s.Edit(func(b *board.State) error {
				b.Cards = []board.Card{card}
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			if tc.locked {
				// 更新前から動いているワーカーと同じ名前でロックを保持する。
				lock, err := s.Lock("worker-7.lock", false)
				if err != nil {
					t.Fatal(err)
				}
				defer lock.Close()
			}

			err := Recover(s, card.ID)
			want := card
			if tc.wantErr == "" {
				if err != nil {
					t.Fatal(err)
				}
				want.Status, want.Result = board.Failed, "中断した実行を手動で復旧しました"
			} else if err == nil || err.Error() != tc.wantErr {
				t.Fatalf("復旧時のエラー = %v, 期待値 = %q", err, tc.wantErr)
			}
			b, err := s.Read()
			if err != nil {
				t.Fatal(err)
			}
			if len(b.Cards) != 1 || b.Cards[0] != want {
				t.Fatalf("復旧後のカード = %+v, 期待値 = %+v", b.Cards, want)
			}
		})
	}
}

func TestWorkerRecordsCodexOutcome(t *testing.T) {
	for _, tc := range []struct {
		name, script, status, result string
	}{
		{"no final message", "exit 0", board.Done, "終了コード 0（最終メッセージなし）"},
		{"interrupted", "exit 130", board.Canceled, ""},
		{"SIGINT with final message", "echo ' partial result ' > \"$3\"\nkill -INT $$", board.Canceled, "partial result"},
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
