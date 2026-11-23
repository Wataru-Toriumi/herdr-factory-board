package worker

import (
	"errors"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/store"
)

func ApproveResult(s store.Store, seen board.Card) error {
	return s.EditCard(seen.ID, func(c *board.Card) error {
		if *c != seen || !c.CanReviewResult() {
			return errors.New("カードが更新されました。最新の実行結果を確認してください")
		}
		c.Status = board.Done
		return nil
	})
}

func Reopen(s store.Store, seen board.Card) error {
	return s.EditCard(seen.ID, func(c *board.Card) error {
		if *c != seen || !c.Reopen() {
			return errors.New("カードを戻せません。最新の状態を確認してください")
		}
		return nil
	})
}

// レビューで表示した版と保存済みのカードをロック内で照合し、別の設計案の承認を防ぐ。
func StartPhase(s store.Store, herdr string, seen board.Card, phase, feedback string) error {
	return launchCard(s, herdr, seen.ID, func(c *board.Card) error {
		if *c != seen {
			return errors.New("カードが更新されました。最新の詳細を確認してください")
		}
		return c.PreparePhase(phase, feedback)
	})
}
