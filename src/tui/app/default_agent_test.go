package app

import (
	"testing"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/schedule"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/store"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestWorkspaceDefaultAgent(t *testing.T) {
	s := store.Store{Dir: t.TempDir()}
	u := newBoardUI(s, "true")
	press := func(key tcell.Key) {
		e := tcell.NewEventKey(key, 0, 0)
		if e = u.key(e); e != nil {
			u.app.GetFocus().InputHandler()(e, func(p tview.Primitive) { u.app.SetFocus(p) })
		}
	}
	press(tcell.KeyCtrlG)
	press(tcell.KeyRight)
	press(tcell.KeyEnter)
	if u.activePage() != "tasks" {
		t.Fatal("保存後に一覧へ戻れません")
	}

	u = newBoardUI(s, "true")
	u.refresh()
	u.workDir = s.Dir
	press(tcell.KeyCtrlG)
	if u.app.GetFocus().(*tview.Button).GetLabel() != "Claude Code" {
		t.Fatal("再起動後に既定の担当が復元されません")
	}
	press(tcell.KeyLeft)
	press(tcell.KeyEscape)

	u.newTaskForm()
	taskPromptField(u).SetText("既定で作成", true)
	press(tcell.KeyCtrlS)
	u.newTaskForm()
	press(tcell.KeyCtrlG)
	taskPromptField(u).SetText("個別に変更", true)
	press(tcell.KeyCtrlS)
	u.newScheduleForm(schedule.Every)
	scheduleTimeField(u).SetText("15m")
	schedulePromptField(u).SetText("定期実行", true)
	press(tcell.KeyCtrlS)

	press(tcell.KeyCtrlG)
	press(tcell.KeyLeft)
	press(tcell.KeyEnter)
	b, err := s.Read()
	if err != nil {
		t.Fatal(err)
	}
	if b.DefaultAgent != board.Codex || len(b.Cards) != 2 || b.Cards[0].Agent != board.Claude || b.Cards[1].Agent != board.Codex ||
		len(b.Schedules) != 1 || b.Schedules[0].Agent != board.Claude {
		t.Fatalf("既定の担当と登録後の担当: %+v", b)
	}
}
