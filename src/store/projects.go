package store

import (
	"errors"
	"os"
	"path/filepath"
	"slices"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/board"
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
