package command

import (
	"testing"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/store"
)

func TestAddAndResultCommands(t *testing.T) {
	s := store.Store{Dir: t.TempDir()}
	if _, err := Parse("dir "+s.Dir).Run(s, "false", s.Dir); err != nil {
		t.Fatal(err)
	}
	if _, err := Parse("add 手動タスク").Run(s, "false", s.Dir); err != nil {
		t.Fatal(err)
	}
	c, err := Parse("result 1").Run(s, "false", s.Dir)
	if err != nil || c == nil || c.ID != 1 || c.Status != board.Todo || c.Dir != s.Dir || c.Prompt != "手動タスク" || c.Result != "" {
		t.Fatalf("結果のカード = %+v, %v", c, err)
	}
}

func TestDailyCommandSavesBeforeSchedulerLaunch(t *testing.T) {
	s := store.Store{Dir: t.TempDir()}
	if _, err := Parse("dir "+s.Dir).Run(s, "false", s.Dir); err != nil {
		t.Fatal(err)
	}
	if _, err := Parse("daily 09:00 朝のタスク").Run(s, "false", s.Dir); err == nil {
		t.Fatal("スケジューラーの起動エラーが返りませんでした")
	}
	b, err := s.Read()
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Schedules) != 1 || len(b.Cards) != 0 {
		t.Fatalf("登録後のボード: %+v", b)
	}
	r := b.Schedules[0]
	if r.Time != "09:00" || r.Prompt != "朝のタスク" || r.Dir != s.Dir {
		t.Fatalf("登録内容: %+v", r)
	}
}
