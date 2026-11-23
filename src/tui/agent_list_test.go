package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/store"
	"github.com/gdamore/tcell/v2"
)

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
	u.refresh()
	u.key(tcell.NewEventKey(tcell.KeyTab, 0, 0))
	u.agents.SetCurrentItem(1)

	t.Setenv("PANE_STATUS", "blocked")
	u.refresh()
	if u.agents.GetCurrentItem() != 0 || !strings.Contains(u.header.GetText(false), "入力待ち 1件") {
		t.Fatalf("更新後の選択行とヘッダー: %d, %s", u.agents.GetCurrentItem(), u.header.GetText(false))
	}
	u.key(tcell.NewEventKey(tcell.KeyEnter, 0, 0))
	got, err := os.ReadFile(os.Getenv("FOCUS_LOG"))
	if err != nil || string(got) != "p2\n" {
		t.Fatalf("移動先のペイン = %q, %v", got, err)
	}
}
