package worker

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

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
		{board.Running, true, "ワーカーまたは担当エージェントが実行中です。終了後に recover を再試行してください"},
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
			if len(b.Cards) != 1 || !b.Cards[0].Equal(want) {
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

func TestRecoveryWaitsForOrphanedAgent(t *testing.T) {
	if dir := os.Getenv("FACTORY_TEST_WORKER_DIR"); dir != "" {
		if err := Run(store.Store{Dir: dir}); err != nil {
			t.Fatal(err)
		}
		return
	}
	s := store.Store{Dir: t.TempDir()}
	if err := s.Edit(func(b *board.State) error {
		b.NewCard("作業", s.Dir, "").QueueAttempt()
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	// CLI が独自に記述子を閉じても、待機中のプロセスはロックを保持する。
	script := `#!/bin/sh
exec 3>&-
touch started
i=0
while [ "$i" -lt 200 ]; do
	i=$((i + 1))
	test -f finish && exit 0
	sleep 0.05
done
exit 1
`
	if err := os.WriteFile(filepath.Join(s.Dir, "codex"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", s.Dir+":"+os.Getenv("PATH"))
	t.Setenv("FACTORY_CARD", "1")
	t.Setenv("FACTORY_ATTEMPT", "1")
	cmd := exec.Command(os.Args[0], "-test.run=^TestRecoveryWaitsForOrphanedAgent$")
	cmd.Env = append(os.Environ(), "FACTORY_TEST_WORKER_DIR="+s.Dir)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.WriteFile(filepath.Join(s.Dir, "finish"), nil, 0600)
		if cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(filepath.Join(s.Dir, "started")); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("エージェントが起動しませんでした")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
	if err := Recover(s, 1); err == nil || err.Error() != "ワーカーまたは担当エージェントが実行中です。終了後に recover を再試行してください" {
		t.Fatalf("エージェント生存中の復旧: %v", err)
	}
	if err := os.WriteFile(filepath.Join(s.Dir, "finish"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	deadline = time.Now().Add(5 * time.Second)
	for {
		err := Recover(s, 1)
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("終了後の復旧: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := Start(s, "true", 1); err != nil {
		t.Fatalf("復旧後の再試行: %v", err)
	}
}
