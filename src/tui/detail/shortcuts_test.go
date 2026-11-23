package detail

import (
	"slices"
	"testing"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
)

func TestReviewApprovalShortcut(t *testing.T) {
	for _, tc := range []struct {
		card board.Card
		want Shortcut
	}{
		{board.Card{Status: board.AwaitingDesignReview, Phase: board.DesignPhase, Design: "設計案"}, Shortcut{'a', "承認して実装", StartImplementation}},
		{board.Card{Status: board.AwaitingResultReview, Phase: board.ImplementationPhase}, Shortcut{'a', "結果を承認して完了", ApproveResult}},
	} {
		shortcuts := Shortcuts(&tc.card, false)
		index := slices.IndexFunc(shortcuts, func(s Shortcut) bool { return s.Key == 'a' })
		if index < 0 || shortcuts[index] != tc.want {
			t.Errorf("%s の承認操作 = %+v, 期待値 = %+v", tc.card.Status, shortcuts, tc.want)
		}
	}
}
