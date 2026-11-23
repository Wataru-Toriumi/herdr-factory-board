package agentlist

import (
	"testing"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/herdr"
)

func TestBoardAgents(t *testing.T) {
	panes := []herdr.Pane{
		{ID: "w1:p1", Workspace: "w1", Label: "Shell"},
		{ID: "w1:p2", Workspace: "w1", Label: "Factory Board Worker"},
		{ID: "w2:p1", Workspace: "w2", Agent: "claude", Status: "blocked", Title: "レビュー"},
		{ID: "w1:p3", Workspace: "w1", Agent: "codex", Status: "blocked"},
	}
	cards := []board.Card{
		{ID: 1, Status: board.Done, Pane: "w1:p1"},
		{ID: 2, Status: board.Running, Pane: "w1:p2", Prompt: "実装"},
		{ID: 3, Status: board.Running, Pane: "w1:p3", Prompt: "確認"},
		{ID: 4, Status: board.Running, Pane: "w1:p4", Prompt: "閉じたペイン"},
	}
	model := Build(panes, []herdr.Workspace{{ID: "w1", Label: "作業場"}}, cards)
	rows := model.Rows
	if model.Blocked != 2 {
		t.Fatalf("入力待ち件数 = %d", model.Blocked)
	}
	if len(rows) != 3 {
		t.Fatalf("一覧: %+v", rows)
	}
	if rows[0].name != "claude" || rows[0].CardID != 0 || rows[0].workspace != "w2" || rows[0].task != "レビュー" {
		t.Errorf("直接起動: %+v", rows[0])
	}
	if rows[1].status != "blocked" || rows[1].CardID != 3 {
		t.Errorf("入力待ち: %+v", rows[1])
	}
	if rows[2].name != "Codex" || rows[2].CardID != 2 || rows[2].workspace != "作業場" || rows[2].task != "#2 実装" {
		t.Errorf("担当カード: %+v", rows[2])
	}
}
