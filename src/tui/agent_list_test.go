package tui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/herdr"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/store"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/tui/agentlist"
	"github.com/gdamore/tcell/v2"
)

func TestAgentEnterUsesCurrentCardPane(t *testing.T) {
	s := store.Store{Dir: t.TempDir()}
	bin := filepath.Join(s.Dir, "herdr")
	log := filepath.Join(s.Dir, "focus")
	t.Setenv("FOCUS_LOG", log)
	if err := os.WriteFile(bin, []byte("#!/bin/sh\necho \"$@\" >> \"$FOCUS_LOG\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	card := board.Card{ID: 1, Status: board.Running, Pane: "p1"}
	u := newBoardUI(s, bin)
	u.agents.Render(agentlist.Build([]herdr.Pane{{ID: "p1"}}, nil, []board.Card{card}))
	u.key(tcell.NewEventKey(tcell.KeyTab, 0, 0))

	// 一覧の次回更新より先に、保存済みのカードが変わる場合を再現する。
	for _, tc := range []struct{ status, pane, notice string }{
		{board.Running, "p2", ""},
		{board.Done, "p2", "実行中の担当ペインがありません。結果は result で確認できます"},
		{board.Running, "", "実行中の担当ペインがありません。結果は result で確認できます"},
	} {
		card.Status, card.Pane = tc.status, tc.pane
		if err := s.Edit(func(b *board.State) error {
			b.Cards = []board.Card{card}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		u.notice.SetText("")
		u.key(tcell.NewEventKey(tcell.KeyEnter, 0, 0))
		if got := u.notice.GetText(false); got != tc.notice {
			t.Fatalf("状態 %q、ペイン %q の通知 = %q", tc.status, tc.pane, got)
		}
	}
	got, err := os.ReadFile(log)
	if err != nil || string(got) != "plugin pane focus p2\n" {
		t.Fatalf("呼び出した Herdr コマンド = %q, %v", got, err)
	}
}

func TestAgentSelectionFollowsPaneAfterRefresh(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "herdr")
	t.Setenv("FOCUS_LOG", filepath.Join(dir, "focus"))
	script := `#!/bin/sh
case "$1 $2" in
    "pane list") printf '{"result":{"panes":[{"pane_id":"p1","agent":"codex"},{"pane_id":"p2","agent":"codex","agent_status":"%s"}]}}' "$PANE_STATUS" ;;
    "workspace list") echo '{"result":{"workspaces":[]}}' ;;
    "agent focus") echo "$3" >> "$FOCUS_LOG" ;;
    *) exit 1 ;;
esac
`
	if err := os.WriteFile(bin, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	u := newBoardUI(store.Store{Dir: dir}, bin)
	t.Setenv("PANE_STATUS", "working")
	u.applyRefresh(readBoard(u.store), readSession(context.Background(), bin))
	u.key(tcell.NewEventKey(tcell.KeyTab, 0, 0))
	u.agents.SetCurrentItem(1)

	t.Setenv("PANE_STATUS", "blocked")
	u.applyRefresh(readBoard(u.store), readSession(context.Background(), bin))
	if u.agents.GetCurrentItem() != 0 || !strings.Contains(u.header.GetText(false), "入力待ち 1件") {
		t.Fatalf("更新後の選択行とヘッダー: %d, %s", u.agents.GetCurrentItem(), u.header.GetText(false))
	}
	u.key(tcell.NewEventKey(tcell.KeyEnter, 0, 0))
	got, err := os.ReadFile(os.Getenv("FOCUS_LOG"))
	if err != nil || string(got) != "p2\n" {
		t.Fatalf("移動先のペイン = %q, %v", got, err)
	}
}
