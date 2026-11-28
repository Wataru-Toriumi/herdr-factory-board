package board

import "testing"

func TestPreparePhaseRejectsBlankReviewFeedback(t *testing.T) {
	for _, tc := range []struct {
		card Card
		want string
	}{
		{Card{Status: AwaitingDesignReview, Phase: DesignPhase, Design: "設計案", Feedback: "前のコメント"}, "再設計のコメントを入力してください"},
		{Card{Status: AwaitingResultReview, Phase: ImplementationPhase, Result: "実装結果", PreviousResult: "前の結果", ImplementationFeedback: "前の指摘"}, "実装の修正内容を入力してください"},
	} {
		t.Run(tc.card.Phase, func(t *testing.T) {
			card := tc.card
			err := card.PreparePhase(card.Phase, " \t\n")
			if err == nil || err.Error() != tc.want {
				t.Fatalf("エラー = %v, 期待値 = %q", err, tc.want)
			}
			if !card.Equal(tc.card) {
				t.Fatalf("コメント拒否後にカードが変更されました: %+v", card)
			}
		})
	}
}
