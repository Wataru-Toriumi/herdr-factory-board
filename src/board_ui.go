package main

import (
	"fmt"
	"strings"
	"time"

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
	failedLane
)

var laneNames = [...]string{"未着手", "設計中", "設計完了", "進行中", "レビュー中", "完了", "失敗"}
var laneColors = [...]tcell.Color{tcell.ColorLightSkyBlue, tcell.ColorPlum, tcell.ColorLavender, tcell.ColorGold, tcell.ColorLightCyan, tcell.ColorLightGreen, tcell.ColorLightCoral}

func cardLane(c card) int {
	switch c.Status {
	case todo:
		return todoLane
	case awaitingReview:
		return designCompleteLane
	case starting, running:
		if c.Phase == designPhase {
			return designLane
		}
		return workLane
	case awaitingResultReview:
		return reviewLane
	case done:
		return doneLane
	case failed:
		return failedLane
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
	store                                 store
	herdr                                 string
	pages                                 *tview.Pages
	columns                               *tview.Flex
	lists                                 []*tview.List
	cards                                 [][]card
	agents                                *tview.List
	schedules                             *scheduleList
	rows                                  []agentRow
	header, context, hint, notice, detail *tview.TextView
	input                                 *tview.InputField
	footer                                *tview.Pages
	workDir                               string
	editingScheduleID                     int
	detailScheduleID                      int
	detailBody                            string
	detailCard                            *card
	editingFeedback                       bool
	spaces                                []workspace
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

func newBoardUI(s store, herdr string) *boardUI {
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
		agents:    boardList(),
		schedules: &scheduleList{List: boardList()},
		cards:     make([][]card, len(laneNames)),
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

func runBoard(s store, herdr string) error {
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
		u.hint.SetText("n 毎日  i 間隔  t 変更  p 作業先  ↑↓ 選択  Enter 詳細  x 解除  Tab 切替  ? ヘルプ")
	default:
		u.pages.SwitchToPage("tasks")
		u.app.SetFocus(u.lists[u.lane])
		u.hint.SetText("v 詳細  n 追加  p 作業先  ? ヘルプ  Tab 切替  ←→ 列  ↑↓ 選択  Enter 操作  q 閉じる")
	}
	u.updateHeader()
}

func (u *boardUI) updateHeader() {
	blocked := 0
	for _, r := range u.rows {
		if r.status == "blocked" {
			blocked++
		}
	}
	labels := tabLabels
	labels[u.tab] = "[black:aqua:b] " + labels[u.tab] + " [-:-:-]"
	u.header.SetText(fmt.Sprintf("[white::b]Factory Board[-:-:-]   %s   [yellow]入力待ち %d件[-]", strings.Join(labels[:], "  "), blocked))
}

func (u *boardUI) refresh() {
	b, err := u.store.read()
	if err != nil {
		u.notice.SetText(err.Error())
		return
	}
	active, err := schedulerRunning(u.store)
	u.workDir = b.Dir
	u.schedules.render(b.Schedules, active, err)
	if u.activePage() == "detail" {
		if u.detailCard != nil {
			if c, err := b.find(u.detailCard.ID); err == nil && *c != *u.detailCard {
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

	panes, err := listPanes(u.herdr)
	if err != nil {
		u.notice.SetText("ペイン一覧: " + err.Error())
		return
	}
	spaces, err := listWorkspaces(u.herdr)
	if err != nil {
		u.notice.SetText("ワークスペース一覧: " + err.Error())
	} else {
		u.spaces = spaces
	}
	u.renderAgents(boardAgents(panes, u.spaces, b.Cards))
}

func (u *boardUI) selectedCard() (card, bool) {
	i := u.lists[u.lane].GetCurrentItem()
	group := u.cards[u.lane]
	if i < 0 || i >= len(group) {
		return card{}, false
	}
	return group[i], true
}

func (u *boardUI) showWorkDir() {
	if u.activePage() == "directory" {
		return
	}
	if u.workDir == "" {
		u.context.SetText("p で作業先を選択")
	} else {
		u.context.SetText("作業先: " + u.workDir)
	}
}
