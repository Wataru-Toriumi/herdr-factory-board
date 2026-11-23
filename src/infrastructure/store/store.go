// Package store はワークスペースごとのボードの保存と排他制御を担当する。
package store

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
)

type Store struct{ Dir string }

func ForWorkspace() (Store, error) {
	root, ws, socket := os.Getenv("HERDR_PLUGIN_STATE_DIR"), os.Getenv("HERDR_WORKSPACE_ID"), os.Getenv("HERDR_SOCKET_PATH")
	if root == "" || ws == "" || socket == "" {
		return Store{}, errors.New("Herdr のプラグインペインから起動してください")
	}
	key := sha256.Sum256([]byte(socket + "\x00" + ws))
	dir := filepath.Join(root, fmt.Sprintf("%x", key[:16]))
	return Store{Dir: dir}, os.MkdirAll(dir, 0700)
}

func (s Store) Read() (board.State, error) {
	var b board.State
	data, err := os.ReadFile(filepath.Join(s.Dir, "board.json"))
	if errors.Is(err, os.ErrNotExist) {
		return b, nil
	}
	if err == nil {
		err = json.Unmarshal(data, &b)
	}
	return b, err
}

// board.json を rename で置き換えても同じ inode をロックできるよう、専用のロックファイルを使う。
func (s Store) Lock(name string, nonblocking bool) (*os.File, error) {
	f, err := os.OpenFile(filepath.Join(s.Dir, name), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	mode := syscall.LOCK_EX
	if nonblocking {
		mode |= syscall.LOCK_NB
	}
	if err := syscall.Flock(int(f.Fd()), mode); err != nil {
		f.Close()
		return nil, err
	}
	return f, nil
}

func (s Store) Edit(change func(*board.State) error) error {
	l, err := s.Lock("board.lock", false)
	if err != nil {
		return err
	}
	defer l.Close()
	b, err := s.Read()
	if err != nil {
		return err
	}
	if err := change(&b); err != nil {
		return err
	}
	f, err := os.CreateTemp(s.Dir, ".board-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	err = json.NewEncoder(f).Encode(&b)
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(f.Name(), filepath.Join(s.Dir, "board.json"))
}

func (s Store) EditCard(id int, change func(*board.Card) error) error {
	return s.Edit(func(b *board.State) error {
		c, err := b.FindCard(id)
		if err != nil {
			return err
		}
		return change(c)
	})
}
