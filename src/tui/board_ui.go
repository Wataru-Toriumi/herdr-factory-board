// Package tui は Factory Board の端末画面とキー操作を担当する。
package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/application/scheduler"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/herdr"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/store"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	todoLane = iota
	designLane
	designCompleteLane
	workLane
	reviewLane
	doneLane
	failedOrCanceledLane
)

var laneNames = [...]string{"未着手", "設計中", "設計完了", "進行中", "レビュー中", "完了", "失敗・中止"}
var laneColors = [...]tcell.Color{tcell.ColorLightSkyBlue, tcell.ColorPlum, tcell.ColorLavender, tcell.ColorGold, tcell.ColorLightCyan, tcell.ColorLightGreen, tcell.ColorLightCoral}

func cardLane(c board.Card) int {
	switch c.Status {
	case board.Todo:
		return todoLane
	case board.AwaitingDesignReview:
		return designCompleteLane
	case board.Starting, board.Running:
		if c.Phase == board.DesignPhase {
			return designLane
		}
		return workLane
	case board.AwaitingResultReview:
		return reviewLane
	case board.Done:
		return doneLane
	case board.Failed, board.Canceled:
		return failedOrCanceledLane
	default:
		return -1
	}
}

const (
	taskTab = iota
	agentTab
	automationTab
)

var tabLabels = [...]string{"タスク", "エージェント", "自動化"}

type boardUI struct {
	app                                   *tview.Application
	store                                 store.Store
	herdr                                 string
	pages                                 *tview.Pages
	columns                               *tview.Flex
	lists                                 []*tview.List
	cards                                 [][]board.Card
	agents                                *agentList
	schedules                             *scheduleList
	header, context, hint, notice, detail *tview.TextView
	input                                 *tview.InputField
	footer                                *tview.Pages
	workDir, inputDir, inputAgent         string
	projects                              []string
	taskProjectNames                      map[string]string
	editingScheduleID                     int
	detailScheduleID                      int
	detailBody                            string
	detailCard                            *board.Card
	editingFeedback                       bool
	spaces                                []herdr.Workspace
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
		agents:    &agentList{List: boardList()},
		schedules: &scheduleList{List: boardList()},
		cards:     make([][]board.Card, len(laneNames)),
	}
	for _, color := range laneColors {
		l := boardList()
		l.SetTitleColor(color)
		u.lists = append(u.lists, l)
	}
	u.layoutColumns()
	u.agents.SetTitle(" エージェント ").SetTitleAlign(tview.AlignLeft)
	u.detail.SetBorder(true).SetTitleAlign(tview.AlignLeft)
	u.pages.AddPage("tasks", u.columns, true, true).AddPage("agents", u.agents, true, false).AddPage("automation", u.schedules, true, false).AddPage("detail", u.detail, true, false)
	u.footer.AddPage("keys", u.hint, true, true).AddPage("input", u.input, true, false)
	root := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(u.header, 1, 0, false).AddItem(u.context, 1, 0, false).
		AddItem(u.pages, 0, 1, true).AddItem(u.notice, 1, 0, false).AddItem(u.footer, 1, 0, false)
	u.app.SetRoot(root, true).SetInputCapture(u.key)
	u.app.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
		width, _ := screen.Size()
		if narrow := width < 17*len(laneNames); narrow != u.compact {
			u.compact = narrow
			u.layoutColumns()
		}
		return false
	})
	u.switchView()
	return u
}

func Run(s store.Store, herdr string) error {
	u := newBoardUI(s, herdr)
	u.refresh()
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				u.app.QueueUpdateDraw(u.refresh)
			}
		}
	}()
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

func (u *boardUI) switchView() {
	u.editingScheduleID = 0
	u.inputDir, u.inputAgent = "", ""
	u.detailScheduleID = 0
	u.detailBody = ""
	u.detailCard = nil
	u.editingFeedback = false
	u.footer.SwitchToPage("keys")
	switch u.tab {
	case agentTab:
		u.pages.SwitchToPage("agents")
		u.app.SetFocus(u.agents)
		u.hint.SetText("Tab 切替   ↑↓ 選択   Enter ペインへ   ? ヘルプ   q 閉じる")
	case automationTab:
		u.pages.SwitchToPage("automation")
		u.app.SetFocus(u.schedules)
		u.hint.SetText("n 毎日  i 間隔  t 変更  p プロジェクト  ↑↓ 選択  Enter 詳細  x 解除  Tab 切替  ? ヘルプ")
	default:
		u.pages.SwitchToPage("tasks")
		u.app.SetFocus(u.lists[u.lane])
		u.hint.SetText("v 詳細  n 追加  p プロジェクト  ? ヘルプ  Tab 切替  ←→ 列  ↑↓ 選択  Enter 操作  q 閉じる")
	}
	u.updateHeader()
	u.showWorkDir()
}

func (u *boardUI) updateHeader() {
	labels := tabLabels
	labels[u.tab] = "[black:aqua:b] " + labels[u.tab] + " [-:-:-]"
	u.header.SetText(fmt.Sprintf("[white::b]Factory Board[-:-:-]   %s   [yellow]入力待ち %d件[-]", strings.Join(labels[:], "  "), u.agents.blockedCount()))
}

func (u *boardUI) refresh() {
	b, err := u.store.Read()
	if err != nil {
		u.notice.SetText(err.Error())
		return
	}
	active, err := scheduler.Running(u.store)
	u.projects = b.ProjectDirs()
	u.schedules.render(b.Schedules, u.workDir, active, err)
	if u.activePage() == "detail" {
		if u.detailCard != nil {
			if c, err := b.FindCard(u.detailCard.ID); err == nil && *c != *u.detailCard {
				u.renderCardDetail(*c)
			}
		}
		u.updateDetailTiming()
	}
	u.renderTasks(b.Cards)
	if err != nil {
		u.notice.SetText(err.Error())
	}
	u.showWorkDir()

	panes, err := herdr.ListPanes(u.herdr)
	if err != nil {
		u.notice.SetText("ペイン一覧: " + err.Error())
		return
	}
	spaces, err := herdr.ListWorkspaces(u.herdr)
	if err != nil {
		u.notice.SetText("ワークスペース一覧: " + err.Error())
	} else {
		u.spaces = spaces
	}
	u.agents.render(boardAgents(panes, u.spaces, b.Cards))
	u.updateHeader()
}

func (u *boardUI) selectedCard() (board.Card, bool) {
	i := u.lists[u.lane].GetCurrentItem()
	group := u.cards[u.lane]
	if i < 0 || i >= len(group) {
		return board.Card{}, false
	}
	return group[i], true
}

func (u *boardUI) showWorkDir() {
	if u.activePage() == "directory" {
		return
	}
	if u.inputDir != "" {
		u.context.SetText("担当: " + board.AgentLabel(u.inputAgent) + "（Ctrl+G 切替）  登録先: " + u.inputDir)
	} else if u.tab == agentTab {
		u.context.SetText("セッション内のエージェント")
	} else if u.workDir == "" {
		u.context.SetText("プロジェクト: すべて   p で選択")
	} else {
		u.context.SetText("プロジェクト: " + u.workDir)
	}
}
