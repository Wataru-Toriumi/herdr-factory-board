package main

import (
	"testing"
)

func TestAddAndResultCommands(t *testing.T) {
	s := store{t.TempDir()}
	if _, err := command(s, "false", s.dir, "dir "+s.dir); err != nil {
		t.Fatal(err)
	}
	if _, err := command(s, "false", s.dir, "add 手動タスク"); err != nil {
		t.Fatal(err)
	}
	c, err := command(s, "false", s.dir, "result 1")
	if err != nil || c == nil || c.ID != 1 || c.Status != todo || c.Dir != s.dir || c.Prompt != "手動タスク" || c.Result != "" {
		t.Fatalf("結果のカード = %+v, %v", c, err)
	}
}

func TestDailyCommandSavesBeforeSchedulerLaunch(t *testing.T) {
	s := store{t.TempDir()}
	if _, err := command(s, "false", s.dir, "dir "+s.dir); err != nil {
		t.Fatal(err)
	}
	if _, err := command(s, "false", s.dir, "daily 09:00 朝のタスク"); err == nil {
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
