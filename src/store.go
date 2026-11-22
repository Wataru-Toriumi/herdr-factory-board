package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

const (
	todo     = "Todo"
	starting = "Starting"
	running  = "In Progress"
	done     = "Done"
	failed   = "Failed"
)

type card struct {
	ID                          int
	Prompt, Dir, Status, Source string
	Attempt                     int
	Pane, Result                string
}

type schedule struct {
	ID                          int
	Time, Prompt, Dir, LastDate string
}

type board struct {
	NextID    int
	Dir       string
	Cards     []card
	Schedules []schedule
}

func (b *board) newCard(prompt, dir, source string) {
	b.NextID++
	b.Cards = append(b.Cards, card{ID: b.NextID, Prompt: prompt, Dir: dir, Source: source, Status: todo})
}

func (b *board) find(id int) (*card, error) {
	for i := range b.Cards {
		if b.Cards[i].ID == id {
			return &b.Cards[i], nil
		}
	}
	return nil, fmt.Errorf("カード #%d がありません", id)
}

type store struct{ dir string }

func workspaceStore() (store, error) {
	root, ws, socket := os.Getenv("HERDR_PLUGIN_STATE_DIR"), os.Getenv("HERDR_WORKSPACE_ID"), os.Getenv("HERDR_SOCKET_PATH")
	if root == "" || ws == "" || socket == "" {
		return store{}, errors.New("Herdr のプラグインペインから起動してください")
	}
	key := sha256.Sum256([]byte(socket + "\x00" + ws))
	dir := filepath.Join(root, fmt.Sprintf("%x", key[:16]))
	return store{dir}, os.MkdirAll(dir, 0700)
}

func (s store) read() (board, error) {
	var b board
	data, err := os.ReadFile(filepath.Join(s.dir, "board.json"))
	if errors.Is(err, os.ErrNotExist) {
		return b, nil
	}
	if err == nil {
		err = json.Unmarshal(data, &b)
	}
	return b, err
}

// board.json を rename で置き換えても同じ inode をロックできるよう、専用のロックファイルを使う。
func (s store) lock(name string, nonblocking bool) (*os.File, error) {
	f, err := os.OpenFile(filepath.Join(s.dir, name), os.O_CREATE|os.O_RDWR, 0600)
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

func (s store) edit(change func(*board) error) error {
	l, err := s.lock("board.lock", false)
	if err != nil {
		return err
	}
	defer l.Close()
	b, err := s.read()
	if err != nil {
		return err
	}
	if err := change(&b); err != nil {
		return err
	}
	f, err := os.CreateTemp(s.dir, ".board-*")
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
	return os.Rename(f.Name(), filepath.Join(s.dir, "board.json"))
}

func (s store) editCard(id int, change func(*card) error) error {
	return s.edit(func(b *board) error {
		c, err := b.find(id)
		if err != nil {
			return err
		}
		return change(c)
	})
}
