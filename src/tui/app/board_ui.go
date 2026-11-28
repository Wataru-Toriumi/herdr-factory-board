// Package app はボードの各画面を組み立て、入力・保存・画面遷移をつなぐ。
package app

import (
	"context"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/dictation"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/herdr"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/store"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/tui/agentlist"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/tui/automation"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/tui/detail"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/tui/taskboard"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/tui/terminal"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type boardUI struct {
	app                           *tview.Application
	store                         store.Store
	herdr                         string
	pages                         *tview.Pages
	tasks, history                *taskboard.View
	showHistory                   bool
	agents                        *agentlist.List
	schedules                     *automation.List
	header, context, hint, notice *tview.TextView
	detail                        *detail.View
	input                         *tview.InputField
	footer                        *tview.Pages
	workDir                       string
	editor                        inputSession
	scheduleForm                  *automation.Form
	taskForm                      *taskboard.Form
	voice                         *voiceDialog
	voiceHelper                   string
	projects                      []string
	board                         boardSnapshot
	panes                         []herdr.Pane
	spaces                        []herdr.Workspace
	sessionError                  string
	tabs                          []tabView
	tab                           int
}

func newBoardUI(s store.Store, herdr string) *boardUI {
	u := &boardUI{
		app:         tview.NewApplication(),
		voiceHelper: dictation.HelperPath(),
		store:       s,
		herdr:       herdr,
		pages:       tview.NewPages(),
		header:      tview.NewTextView().SetDynamicColors(true),
		context:     tview.NewTextView().SetWrap(false),
		notice:      tview.NewTextView().SetWrap(false),
		hint:        tview.NewTextView().SetWrap(false),
		input:       tview.NewInputField().SetLabel("> "),
		footer:      tview.NewPages(),
	}
	u.schedules = u.newScheduleList()
	u.tasks = taskboard.NewView(taskboard.Manual, u.execute, u.focusBoardLane("tasks"))
	u.history = taskboard.NewView(taskboard.RunHistory, u.execute, u.focusBoardLane("history"))
	u.agents = agentlist.NewList(u.focusAgent)
	u.detail = detail.NewView(u.schedules.ByID, func(hint string) { u.hint.SetText(hint) })
	u.registerTabs()
	u.pages.AddPage("detail", u.detail, true, false)
	u.footer.AddPage("keys", u.hint, true, true).AddPage("input", u.input, true, false)
	root := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(u.header, 1, 0, false).AddItem(u.context, 1, 0, false).
		AddItem(u.pages, 0, 1, true).AddItem(u.notice, 1, 0, false).AddItem(u.footer, 1, 0, false)
	u.app.SetRoot(root, true).SetInputCapture(u.key)
	u.app.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
		width, _ := screen.Size()
		u.tasks.Resize(width)
		u.history.Resize(width)
		return false
	})
	u.switchView()
	return u
}

func (u *boardUI) focusBoardLane(page string) func(tview.Primitive) {
	return func(list tview.Primitive) {
		if u.activePage() == page {
			u.app.SetFocus(list)
		}
	}
}

// Run は指定された端末画面でボードの更新と入力受付を開始する。
func Run(s store.Store, herdr string, screen tcell.Screen) error {
	u := newBoardUI(s, herdr)
	if err := terminal.SetScreen(u.app, screen); err != nil {
		return err
	}
	return u.run()
}

func (u *boardUI) run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer func() {
		if u.voice != nil {
			u.voice.session.Cancel()
		}
	}()
	u.refresh()
	go u.poll(ctx)
	return u.app.Run()
}

// コマンド入力中はフッター、それ以外は本文のページがキー操作を受け取る。
func (u *boardUI) activePage() string {
	if page, _ := u.footer.GetFrontPage(); page == "input" {
		return page
	}
	page, _ := u.pages.GetFrontPage()
	return page
}

func (u *boardUI) showWorkDir() {
	switch u.activePage() {
	case "task-form":
		u.context.SetText(u.taskForm.Context())
		return
	case "schedule-form":
		u.context.SetText(u.scheduleForm.Context())
		return
	case "directory", "dictation":
		return
	}
	if u.tab == agentTab {
		u.context.SetText("セッション内のエージェント")
	} else if u.workDir == "" {
		u.context.SetText("プロジェクト: すべて   p で選択")
	} else {
		u.context.SetText("プロジェクト: " + u.workDir)
	}
}
