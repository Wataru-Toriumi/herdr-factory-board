package tui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	taskTab = iota
	agentTab
	automationTab
)

type tabView struct {
	page, label, hint string
	body              tview.Primitive
	handleKey         func(*tcell.EventKey) bool
}

func (u *boardUI) registerTabs() {
	u.tabs = []tabView{
		taskTab: {page: "tasks", label: "タスク", body: u.columns, handleKey: u.taskListKey,
			hint: "v 詳細  n 追加  p プロジェクト  ? ヘルプ  Tab 切替  ←→ 列  ↑↓ 選択  Enter 操作  q 閉じる"},
		agentTab: {page: "agents", label: "エージェント", body: u.agents, handleKey: u.agentListKey,
			hint: "Tab 切替   ↑↓ 選択   Enter ペインへ   ? ヘルプ   q 閉じる"},
		automationTab: {page: "automation", label: "自動化", body: u.schedules, handleKey: u.scheduleListKey,
			hint: "n 毎日  i 間隔  t 変更  p プロジェクト  ↑↓ 選択  Enter 詳細  x 解除  Tab 切替  ? ヘルプ"},
	}
	for i, tab := range u.tabs {
		u.pages.AddPage(tab.page, tab.body, true, i == u.tab)
	}
}

func (u *boardUI) switchView() {
	u.editor = inputSession{}
	u.detailScheduleID = 0
	u.detailBody = ""
	u.detailCard = nil
	u.footer.SwitchToPage("keys")
	tab := u.tabs[u.tab]
	u.pages.SwitchToPage(tab.page)
	focus := tab.body
	if u.tab == taskTab {
		focus = u.lists[u.lane]
	}
	u.app.SetFocus(focus)
	u.hint.SetText(tab.hint)
	u.updateHeader()
	u.showWorkDir()
}

func (u *boardUI) updateHeader() {
	labels := make([]string, len(u.tabs))
	for i, tab := range u.tabs {
		labels[i] = tab.label
	}
	labels[u.tab] = "[black:aqua:b] " + labels[u.tab] + " [-:-:-]"
	u.header.SetText(fmt.Sprintf("[white::b]Factory Board[-:-:-]   %s   [yellow]入力待ち %d件[-]", strings.Join(labels, "  "), u.agents.blocked))
}
