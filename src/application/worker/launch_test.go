package worker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/store"
)

func TestRetryRunsWithEnvironmentPassedToHerdr(t *testing.T) {
	s := store.Store{Dir: t.TempDir()}
	if err := s.Edit(func(b *board.State) error {
		b.Cards = []board.Card{{ID: 17, Attempt: 3, Status: board.Failed, Dir: s.Dir}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	bin := t.TempDir()
	t.Setenv("LAUNCH_ENV", filepath.Join(bin, "env"))
	t.Setenv("PATH", bin+":"+os.Getenv("PATH"))
	herdr := filepath.Join(bin, "herdr")
	script := `#!/bin/sh
while [ "$#" -gt 0 ]; do
	if [ "$1" = "--env" ]; then
		shift
		echo "$1" >> "$LAUNCH_ENV"
	fi
	shift
done
`
	if err := os.WriteFile(herdr, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "codex"), []byte("#!/bin/sh\necho 完了 > \"$3\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := Start(s, herdr, 17); err != nil {
		t.Fatal(err)
	}
	env, err := os.ReadFile(os.Getenv("LAUNCH_ENV"))
	if err != nil || string(env) != "FACTORY_CARD=17\nFACTORY_ATTEMPT=4\n" {
		t.Fatalf("Herdr に渡した環境変数 = %q, %v", env, err)
	}
	for _, assignment := range strings.Fields(string(env)) {
		key, value, _ := strings.Cut(assignment, "=")
		t.Setenv(key, value)
	}
	if err := Run(s); err != nil {
		t.Fatal(err)
	}
	b, err := s.Read()
	if err != nil {
		t.Fatal(err)
	}
	c, err := b.FindCard(17)
	if err != nil || c.Attempt != 4 || c.Status != board.Done || c.Result != "完了" {
		t.Fatalf("再試行後のカード = %+v, %v", c, err)
	}
}

func TestRunRejectsInvalidLaunchEnvironment(t *testing.T) {
	for _, tc := range []struct{ card, attempt, want string }{
		{"", "bad", "ボードから実行してください"},
		{"17", "bad", `strconv.Atoi: parsing "bad": invalid syntax`},
	} {
		t.Setenv("FACTORY_CARD", tc.card)
		t.Setenv("FACTORY_ATTEMPT", tc.attempt)
		if err := Run(store.Store{Dir: t.TempDir()}); err == nil || err.Error() != tc.want {
			t.Fatalf("カード %q、実行番号 %q のエラー = %v", tc.card, tc.attempt, err)
		}
	}
}
