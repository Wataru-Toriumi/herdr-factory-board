package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/store"
	"github.com/gdamore/tcell/v2"
)

func TestAgentSelectionForCardsAndSchedules(t *testing.T) {
	s := store.Store{Dir: t.TempDir()}
	u := newBoardUI(s, "true")
	press := func(k tcell.Key) { u.key(tcell.NewEventKey(k, 0, 0)) }
	key := func(r rune) { u.key(tcell.NewEventKey(tcell.KeyRune, r, 0)) }

	u.editForProject("add ", s.Dir)
	press(tcell.KeyCtrlG)
	u.input.SetText("add 手動タスク")
	press(tcell.KeyEnter)
	key('v')
	if u.detailCard.Agent != board.Claude || !strings.Contains(u.detail.GetText(false), "Claude Code") {
		t.Fatalf("手動カードの担当: %+v", u.detailCard)
	}
	key('g')
	if u.detailCard.Agent != board.Codex {
		t.Fatal("未着手カードの担当を変更できません")
	}
	press(tcell.KeyEscape)
	press(tcell.KeyBacktab)
	key('i')
	press(tcell.KeyCtrlG)
	u.input.SetText("every 1m 定期タスク")
	press(tcell.KeyEnter)
	press(tcell.KeyEnter)
	if !strings.Contains(u.detail.GetText(false), "Claude Code") {
		t.Fatalf("予定の担当: %s", u.detail.GetText(false))
	}
	now := time.Now().Add(2 * time.Minute)
	record := func(at time.Time) {
		t.Helper()
		if err := s.Edit(func(b *board.State) error { b.RecordDueSchedules(at); return nil }); err != nil {
			t.Fatal(err)
		}
	}
	record(now)
	key('g')
	record(now.Add(time.Minute))
	b, err := s.Read()
	if err != nil || len(b.Cards) != 3 || b.Cards[1].Agent != board.Claude || b.Cards[2].Agent != board.Codex {
		t.Fatalf("担当変更前後の履歴: %+v, %v", b.Cards, err)
	}
}
