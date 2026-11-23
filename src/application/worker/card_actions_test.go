package worker

import (
	"testing"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/store"
)

func TestCardActionsRejectStaleResult(t *testing.T) {
	for _, tc := range []struct {
		name, status, message string
		act                   func(store.Store, board.Card) error
	}{
		{"キャンセル", board.Running, "カードが更新されました。最新の状態を確認してください", Cancel},
		{"結果の承認", board.AwaitingResultReview, "カードが更新されました。最新の実行結果を確認してください", ApproveResult},
		{"再オープン", board.Done, "カードを戻せません。最新の状態を確認してください", Reopen},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := store.Store{Dir: t.TempDir()}
			seen := board.Card{ID: 1, Status: tc.status, Phase: board.ImplementationPhase, Result: "表示時の結果"}
			current := seen
			current.Result = "最新の結果"
			if err := s.Edit(func(b *board.State) error {
				b.Cards = []board.Card{current}
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			if err := tc.act(s, seen); err == nil || err.Error() != tc.message {
				t.Fatalf("競合時のエラー = %v, 期待値 = %q", err, tc.message)
			}
			b, err := s.Read()
			if err != nil {
				t.Fatal(err)
			}
			if len(b.Cards) != 1 || b.Cards[0] != current {
				t.Fatalf("競合後のカード = %+v", b.Cards)
			}
		})
	}
}
