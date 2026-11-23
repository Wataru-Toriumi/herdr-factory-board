package main

import (
	"testing"
)

func TestAddAndResultCommands(t *testing.T) {
	s := store{t.TempDir()}
	if _, err := parseCommand("dir "+s.dir).run(s, "false", s.dir); err != nil {
		t.Fatal(err)
	}
	if _, err := parseCommand("add 手動タスク").run(s, "false", s.dir); err != nil {
		t.Fatal(err)
	}
	c, err := parseCommand("result 1").run(s, "false", s.dir)
	if err != nil || c == nil || c.ID != 1 || c.Status != todo || c.Dir != s.dir || c.Prompt != "手動タスク" || c.Result != "" {
		t.Fatalf("結果のカード = %+v, %v", c, err)
	}
}

func TestDailyCommandSavesBeforeSchedulerLaunch(t *testing.T) {
	s := store{t.TempDir()}
	if _, err := parseCommand("dir "+s.dir).run(s, "false", s.dir); err != nil {
		t.Fatal(err)
	}
	if _, err := parseCommand("daily 09:00 朝のタスク").run(s, "false", s.dir); err == nil {
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
