package main

import (
	"fmt"
	"testing"
)

func TestAddAndResultCommands(t *testing.T) {
	s := store{t.TempDir()}
	if _, err := command(s, "false", "dir "+s.dir); err != nil {
		t.Fatal(err)
	}
	if _, err := command(s, "false", "add 手動タスク"); err != nil {
		t.Fatal(err)
	}
	text, err := command(s, "false", "result 1")
	if want := fmt.Sprintf("#1  Todo  %s\n手動タスク\n", s.dir); err != nil || text != want {
		t.Fatalf("result = %q, %v; want %q", text, err, want)
	}
}

func TestDailyCommandSavesBeforeSchedulerLaunch(t *testing.T) {
	s := store{t.TempDir()}
	if _, err := command(s, "false", "dir "+s.dir); err != nil {
		t.Fatal(err)
	}
	if _, err := command(s, "false", "daily 09:00 朝のタスク"); err == nil {
		t.Fatal("スケジューラーの起動エラーが返りませんでした")
	}
	b, err := s.read()
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Schedules) != 1 || len(b.Cards) != 0 {
		t.Fatalf("登録後のボード: %+v", b)
	}
	r := b.Schedules[0]
	if r.Time != "09:00" || r.Prompt != "朝のタスク" || r.Dir != s.dir {
		t.Fatalf("登録内容: %+v", r)
	}
}
