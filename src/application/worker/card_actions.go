package worker

import (
	"errors"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/store"
)

func Cancel(s store.Store, seen board.Card) error {
	return applyCardAction(s, seen, (*board.Card).Cancel, "カードが更新されました。最新の状態を確認してください")
}

func ApproveResult(s store.Store, seen board.Card) error {
	return applyCardAction(s, seen, (*board.Card).ApproveResult, "カードが更新されました。最新の実行結果を確認してください")
}

func Reopen(s store.Store, seen board.Card) error {
	return applyCardAction(s, seen, (*board.Card).Reopen, "カードを戻せません。最新の状態を確認してください")
}

// カードの照合と状態変更は、同じ保存ロックの内側で行う。
func applyCardAction(s store.Store, seen board.Card, apply func(*board.Card) bool, rejection string) error {
	return s.EditCard(seen.ID, func(c *board.Card) error {
		if !c.Equal(seen) || !apply(c) {
			return errors.New(rejection)
		}
		return nil
	})
}
