package board

import "testing"

func TestLaunchUpdatesOnlyAffectCurrentStartingAttempt(t *testing.T) {
	for _, tc := range []struct {
		name string
		card Card
	}{
		{"次の実行が起動待ち", Card{Attempt: 2, Status: Starting}},
		{"すでに実行中", Card{Attempt: 1, Status: Running, Pane: "担当ペイン"}},
		{"起動前にキャンセル済み", Card{Attempt: 1, Status: Canceled}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := tc.card
			if card.MarkAttemptRunning(1, "別のペイン") || !card.Equal(tc.card) {
				t.Fatalf("無効な起動要求が反映されました: %+v", card)
			}
			card.FailAttemptLaunch(1, "起動失敗")
			if !card.Equal(tc.card) {
				t.Fatalf("遅れて届いた起動失敗が反映されました: %+v", card)
			}
		})
	}
}

func TestFinishAttemptIgnoresObsoleteResult(t *testing.T) {
	for _, tc := range []struct {
		name string
		card Card
	}{
		{"再試行が実行中", Card{Attempt: 2, Status: Running, Phase: DesignPhase}},
		{"同じ実行をキャンセル済み", Card{Attempt: 1, Status: Canceled, Phase: DesignPhase}},
		{"同じ実行の設計案を保存済み", Card{Attempt: 1, Status: AwaitingDesignReview, Phase: DesignPhase, Design: "保存済みの設計案"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := tc.card
			card.FinishAttempt(1, DesignPhase, Done, "遅れて届いた設計案")
			if !card.Equal(tc.card) {
				t.Fatalf("遅れて届いた結果でカードが変更されました: %+v", card)
			}
		})
	}
}
