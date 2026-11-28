package command

import (
	"strings"
	"testing"
	"time"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/schedule"
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
	_, err := Parse("daily 09:00 朝のタスク").Run(s, "false", s.Dir)
	if err == nil || !strings.Contains(err.Error(), "予定 #1 は保存済み") || !strings.Contains(err.Error(), "scheduler で起動を再試行") {
		t.Fatalf("登録済みの予定と起動の再試行方法: %v", err)
	}
	if _, err := Parse("scheduler").Run(s, "true", s.Dir); err != nil {
		t.Fatal(err)
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

func TestDefaultAgentForCommandsAndLegacySchedule(t *testing.T) {
	s := store.Store{Dir: t.TempDir()}
	if err := s.Edit(func(b *board.State) error {
		b.NewCard("以前のカード", s.Dir, "")
		b.AddSchedule(schedule.Rule{Time: "00:00", Prompt: "以前の予定", Dir: s.Dir})
		b.DefaultAgent = board.Claude
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{"add 新規カード", "daily 09:00 朝のタスク", "every 15m 差分確認"} {
		if _, err := Parse(line).Run(s, "true", s.Dir); err != nil {
			t.Fatal(err)
		}
	}
	b, err := s.Read()
	if err != nil {
		t.Fatal(err)
	}
	if b.Cards[0].Agent != "" || b.Schedules[0].Agent != "" || b.Cards[1].Agent != board.Claude ||
		b.Schedules[1].Agent != board.Claude || b.Schedules[2].Agent != board.Claude {
		t.Fatalf("新規登録後の担当: %+v", b)
	}
	b.RecordDueSchedules(time.Now().AddDate(0, 0, 1))
	for _, c := range b.Cards[2:] {
		if c.ScheduleID == b.Schedules[0].ID && board.AgentLabel(c.Agent) == "Codex" {
			return
		}
	}
	t.Fatal("既存の予定からCodexのカードが生成されませんでした")
}
