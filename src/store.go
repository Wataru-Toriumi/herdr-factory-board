package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	todo                 = "Todo"
	starting             = "Starting"
	running              = "In Progress"
	awaitingReview       = "Awaiting Design Review"
	awaitingResultReview = "Awaiting Result Review"
	done                 = "Done"
	failed               = "Failed"
	canceled             = "Canceled"
)

type card struct {
	ID                          int
	Prompt, Dir, Status, Source string
	Attempt                     int
	Pane, Result                string
	ScheduleID                  int    `json:",omitempty"`
	Phase                       string `json:",omitempty"`
	Design                      string `json:",omitempty"`
	Feedback                    string `json:",omitempty"`
}

func (c card) canCancel() bool {
	switch c.Status {
	case todo, starting, running, awaitingReview, awaitingResultReview, failed:
		return true
	}
	return false
}

func cancelCard(s store, seen card) error {
	return s.editCard(seen.ID, func(c *card) error {
		if c.Attempt != seen.Attempt || c.Status != seen.Status || !c.canCancel() {
			return errors.New("カードが更新されました。最新の状態を確認してください")
		}
		c.Status = canceled
		return nil
	})
}

func (c card) sourceScheduleID() int {
	if c.ScheduleID != 0 {
		return c.ScheduleID
	}
	// 予定番号を独立して保存する前のカードでは、履歴の先頭に「定期 #番号」がある。
	source, ok := strings.CutPrefix(c.Source, "定期 #")
	if !ok {
		return 0
	}
	number, _, _ := strings.Cut(source, " ")
	id, err := strconv.Atoi(number)
	if err != nil || id < 1 {
		return 0
	}
	return id
}

type schedule struct {
	ID                          int
	Time, Prompt, Dir, LastDate string
	Interval                    *scheduleInterval `json:",omitempty"`
}

type scheduleInterval struct {
	Minutes int64
	NextRun time.Time
}

type board struct {
	NextID    int
	Dir       string
	Cards     []card
	Schedules []schedule
	Projects  []string `json:",omitempty"`
}

func (b *board) newCard(prompt, dir, source string) *card {
	b.NextID++
	b.Cards = append(b.Cards, card{ID: b.NextID, Prompt: prompt, Dir: dir, Source: source, Status: todo})
	return &b.Cards[len(b.Cards)-1]
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
