package main

import "testing"

func TestBoardAgents(t *testing.T) {
	panes := []livePane{
		{ID: "w1:p1", Workspace: "w1", Label: "Shell"},
		{ID: "w1:p2", Workspace: "w1", Label: "Factory Board Worker"},
		{ID: "w2:p1", Workspace: "w2", Agent: "claude", Status: "blocked", Title: "レビュー"},
		{ID: "w1:p3", Workspace: "w1", Agent: "codex", Status: "blocked"},
	}
	cards := []card{
		{ID: 1, Status: done, Pane: "w1:p1"},
		{ID: 2, Status: running, Pane: "w1:p2", Prompt: "実装"},
		{ID: 3, Status: running, Pane: "w1:p3", Prompt: "確認"},
		{ID: 4, Status: running, Pane: "w1:p4", Prompt: "閉じたペイン"},
	}
	rows := boardAgents(panes, []workspace{{ID: "w1", Label: "作業場"}}, cards)
	if len(rows) != 3 {
		t.Fatalf("一覧: %+v", rows)
	}
	if rows[0].name != "claude" || rows[0].cardID != 0 || rows[0].workspace != "w2" || rows[0].task != "レビュー" {
		t.Errorf("直接起動: %+v", rows[0])
	}
	if rows[1].status != "blocked" || rows[1].cardID != 3 {
		t.Errorf("入力待ち: %+v", rows[1])
	}
	if rows[2].name != "Codex" || rows[2].cardID != 2 || rows[2].workspace != "作業場" || rows[2].task != "#2 実装" {
		t.Errorf("担当カード: %+v", rows[2])
	}
}
