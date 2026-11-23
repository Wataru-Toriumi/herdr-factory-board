package store

import (
	"errors"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
)

func (s Store) ChangeCardProject(seen board.Card, dir string) error {
	return s.editCardConfig(seen, "チケットが更新されました。最新の作業先と状態を確認してください", func(c *board.Card) {
		c.Dir = dir
	})
}

func (s Store) ChangeCardAgent(seen board.Card, agent string) error {
	return s.editCardConfig(seen, "カードが更新されました。最新の状態を確認してください", func(c *board.Card) {
		c.Agent = agent
	})
}

// 画面に表示したカードと保存済みのカードをロック内で照合し、別ペインでの更新を保護する。
func (s Store) editCardConfig(seen board.Card, conflictMessage string, change func(*board.Card)) error {
	return s.EditCard(seen.ID, func(c *board.Card) error {
		if *c != seen || !c.CanConfigure() {
			return errors.New(conflictMessage)
		}
		change(c)
		return nil
	})
}
