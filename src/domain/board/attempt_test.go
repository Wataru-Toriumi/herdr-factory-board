package board

import "testing"

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
			if card != tc.card {
				t.Fatalf("遅れて届いた結果でカードが変更されました: %+v", card)
			}
		})
	}
}
