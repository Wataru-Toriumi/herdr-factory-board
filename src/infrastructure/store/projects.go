package store

import (
	"errors"
	"os"
	"path/filepath"
	"slices"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
)

func (s Store) RegisterProject(path string) error {
	if !filepath.IsAbs(path) {
		return errors.New("作業先の絶対パスを指定してください")
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("作業先はディレクトリを指定してください")
	}
	path = filepath.Clean(path)
	return s.Edit(func(b *board.State) error {
		if !slices.Contains(b.Projects, path) {
			b.Projects = append(b.Projects, path)
		}
		return nil
	})
}

// ChangeCardProject は作業先の選択中にカードが更新されていないか、ロック内で照合する。
func (s Store) ChangeCardProject(seen board.Card, dir string) error {
	return s.EditCard(seen.ID, func(c *board.Card) error {
		if !c.CanConfigure() || *c != seen {
			return errors.New("チケットが更新されました。最新の作業先と状態を確認してください")
		}
		c.Dir = dir
		return nil
	})
}
