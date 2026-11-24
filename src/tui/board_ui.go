// Package tui は Factory Board の端末画面とキー操作を担当する。
package tui

import (
	"context"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/schedule"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/herdr"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/store"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/tui/agentlist"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/tui/taskboard"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type boardUI struct {
	app                                   *tview.Application
	store                                 store.Store
	herdr                                 string
	pages                                 *tview.Pages
	columns                               *tview.Flex
	lists                                 [len(taskLanes)]*rowList[taskboard.Row]
	agents                                *agentList
	schedules                             *scheduleList
	header, context, hint, notice, detail *tview.TextView
	input                                 *tview.InputField
	footer                                *tview.Pages
	workDir                               string
	editor                                inputSession
	scheduleForm                          *scheduleForm
	projects                              []string
	detailScheduleID                      int
	detailBody                            string
	detailCard                            *board.Card
	board                                 boardSnapshot
	panes                                 []herdr.Pane
	spaces                                []herdr.Workspace
	sessionError                          string
	tabs                                  []tabView
	tab                                   int
	lane                                  int
	compact                               bool
}

func boardList() *tview.List {
	l := tview.NewList().SetSelectedFocusOnly(true).SetHighlightFullLine(true).SetWrapAround(false)
	l.SetSecondaryTextColor(tcell.ColorLightSlateGray)
	l.SetSelectedStyle(tcell.StyleDefault.Background(tcell.ColorDarkCyan).Foreground(tcell.ColorWhite))
	l.SetBorder(true)
	return l
}

func newBoardUI(s store.Store, herdr string) *boardUI {
	u := &boardUI{
		app:       tview.NewApplication(),
		store:     s,
		herdr:     herdr,
		pages:     tview.NewPages(),
		columns:   tview.NewFlex(),
		header:    tview.NewTextView().SetDynamicColors(true),
		context:   tview.NewTextView().SetWrap(false),
		detail:    tview.NewTextView().SetScrollable(true).SetWordWrap(true),
		notice:    tview.NewTextView().SetWrap(false),
		hint:      tview.NewTextView().SetWrap(false),
		input:     tview.NewInputField().SetLabel("> "),
		footer:    tview.NewPages(),
		agents:    &agentList{rowList: newRowList[agentlist.Row]()},
		schedules: &scheduleList{rowList: newRowList[schedule.Rule]()},
	}
	for i, lane := range taskLanes {
		u.lists[i] = newRowList[taskboard.Row]()
		u.lists[i].SetTitleColor(lane.color)
	}
	u.layoutColumns()
	u.agents.SetTitle(" エージェント ").SetTitleAlign(tview.AlignLeft)
	u.detail.SetBorder(true).SetTitleAlign(tview.AlignLeft)
	u.registerTabs()
	u.pages.AddPage("detail", u.detail, true, false)
	u.footer.AddPage("keys", u.hint, true, true).AddPage("input", u.input, true, false)
	root := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(u.header, 1, 0, false).AddItem(u.context, 1, 0, false).
		AddItem(u.pages, 0, 1, true).AddItem(u.notice, 1, 0, false).AddItem(u.footer, 1, 0, false)
	u.app.SetRoot(root, true).SetInputCapture(u.key)
	u.app.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
		width, _ := screen.Size()
		if narrow := width < 17*len(taskLanes); narrow != u.compact {
			u.compact = narrow
			u.layoutColumns()
		}
		return false
	})
	u.switchView()
	return u
}

func Run(s store.Store, herdr string) error {
	screen, err := tcell.NewScreen()
	if err != nil {
		return err
	}
	u := newBoardUI(s, herdr)
	if err := setBoardScreen(u.app, screen); err != nil {
		return err
	}
	return u.run()
}

func (u *boardUI) run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	u.refresh()
	go u.poll(ctx)
	return u.app.Run()
}

func (u *boardUI) layoutColumns() {
	u.columns.Clear()
	for i, l := range u.lists {
		if !u.compact || i == u.lane {
			u.columns.AddItem(l, 0, 1, false)
		}
	}
}

// コマンド入力中はフッター、それ以外は本文のページがキー操作を受け取る。
func (u *boardUI) activePage() string {
	if page, _ := u.footer.GetFrontPage(); page == "input" {
		return page
	}
	page, _ := u.pages.GetFrontPage()
	return page
}

func (u *boardUI) selectedCard() (board.Card, bool) {
	row, ok := u.lists[u.lane].selected()
	return row.Card, ok
}

func (u *boardUI) showWorkDir() {
	if u.activePage() == "schedule-form" {
		f := u.scheduleForm
		u.context.SetText("担当: " + board.AgentLabel(f.agent) + "（Ctrl+G 切替）  作業先: " + f.rule.Dir)
		return
	}
	if u.activePage() == "directory" {
		return
	}
	if u.editor.dir != "" {
		u.context.SetText("担当: " + board.AgentLabel(u.editor.agent) + "（Ctrl+G 切替）  登録先: " + u.editor.dir)
	} else if u.tab == agentTab {
		u.context.SetText("セッション内のエージェント")
	} else if u.workDir == "" {
		u.context.SetText("プロジェクト: すべて   p で選択")
	} else {
		u.context.SetText("プロジェクト: " + u.workDir)
	}
}
