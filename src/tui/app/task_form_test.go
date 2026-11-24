package app

import (
	"strings"
	"testing"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/store"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/tui/terminal"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func taskPromptField(u *boardUI) *tview.TextArea {
	return u.taskForm.GetFormItemByLabel("作業内容").(*tview.TextArea)
}

func TestNewTaskFormPreservesDraftWhileChoosingProject(t *testing.T) {
	s := store.Store{Dir: t.TempDir()}
	t.Setenv("HOME", s.Dir)
	u := newBoardUI(s, "true")
	press := func(key tcell.Key) { u.key(tcell.NewEventKey(key, 0, 0)) }
	open := func() { u.key(tcell.NewEventKey(tcell.KeyRune, 'n', 0)) }
	open()
	draft := "調査する\n\n  手順を記録"
	taskPromptField(u).SetText(draft, true)
	press(tcell.KeyCtrlG)
	press(tcell.KeyCtrlP)
	press(tcell.KeyEscape)
	if u.activePage() != "task-form" || taskPromptField(u).GetText() != draft {
		t.Fatal("作業先の選択を取り消すと下書きが失われます")
	}
	press(tcell.KeyCtrlS)
	page, picker := u.pages.GetFrontPage()
	if page != "directory" {
		t.Fatal("作業先を選択できません")
	}
	picker.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), func(p tview.Primitive) { u.app.SetFocus(p) })
	if u.activePage() != "task-form" || u.taskForm.Project != s.Dir || taskPromptField(u).GetText() != draft {
		t.Fatal("作業先の登録後にフォームへ戻れません")
	}
	taskPromptField(u).SetText(" \n　", true)
	press(tcell.KeyCtrlS)
	if u.activePage() != "task-form" || !strings.Contains(u.notice.GetText(false), "作業内容を入力") {
		t.Fatal("空白だけの作業内容が受理されました")
	}
	taskPromptField(u).SetText(draft, true)
	press(tcell.KeyCtrlS)
	if u.activePage() != "tasks" {
		t.Fatal("保存後にタスク一覧に戻れません")
	}
	open()
	if u.taskForm.Project != s.Dir || taskPromptField(u).GetText() != "" {
		t.Fatal("次の登録で作業先または本文の初期値が不正です")
	}
	taskPromptField(u).SetText("取り消す本文", true)
	press(tcell.KeyEscape)
	b, err := s.Read()
	if err != nil || len(b.Cards) != 1 {
		t.Fatalf("登録後のカード: %+v, %v", b.Cards, err)
	}
	c := b.Cards[0]
	if c.Prompt != draft || c.Agent != board.Claude || c.Dir != s.Dir || c.Status != board.Todo {
		t.Fatalf("新規タスクの設定が失われました: %+v", c)
	}
}

func TestTaskFormKeyboardAndPaste(t *testing.T) {
	s := store.Store{Dir: t.TempDir()}
	u := newBoardUI(s, "true")
	u.workDir = s.Dir
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := terminal.SetScreen(u.app, screen); err != nil {
		t.Fatal(err)
	}
	key := func(k tcell.Key) { screen.PostEventWait(tcell.NewEventKey(k, 0, 0)) }
	typeText := func(text string) {
		for _, r := range text {
			screen.PostEventWait(tcell.NewEventKey(tcell.KeyRune, r, 0))
		}
	}
	go func() {
		typeText("n調査")
		key(tcell.KeyEnter)
		screen.PostEventWait(tcell.NewEventPaste(true))
		typeText("手順")
		key(tcell.KeyCtrlJ)
		typeText("  詳細")
		screen.PostEventWait(tcell.NewEventPaste(false))
		key(tcell.KeyTab)
		key(tcell.KeyEnter)
		key(tcell.KeyDown)
		key(tcell.KeyEnter)
		key(tcell.KeyTab)
		key(tcell.KeyEnter)
		key(tcell.KeyCtrlC)
	}()
	if err := u.app.Run(); err != nil {
		t.Fatal(err)
	}
	b, err := s.Read()
	if err != nil || len(b.Cards) != 1 || b.Cards[0].Prompt != "調査\n手順\n  詳細" || b.Cards[0].Agent != board.Claude {
		t.Fatalf("キー入力から保存したタスク: %+v, %v", b.Cards, err)
	}
}
