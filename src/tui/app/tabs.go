package app

import (
	"fmt"
	"strings"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/tui/agentlist"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/tui/automation"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/tui/taskboard"
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
		taskTab: {page: "tasks", label: "タスク", body: u.tasks, handleKey: u.tasks.HandleKey,
			hint: taskboard.ListHint},
		agentTab: {page: "agents", label: "エージェント", body: u.agents, handleKey: u.agents.HandleKey,
			hint: agentlist.ListHint},
		automationTab: {page: "automation", label: "自動化", body: u.schedules, handleKey: u.schedules.HandleKey,
			hint: "h 実行履歴  " + automation.ListHint},
	}
	u.pages.AddPage("history", u.history, true, false)
	for i, tab := range u.tabs {
		u.pages.AddPage(tab.page, tab.body, true, i == u.tab)
	}
}

func (u *boardUI) currentView() tabView {
	if u.tab == automationTab && u.showHistory {
		return tabView{page: "history", body: u.history, handleKey: u.history.HandleKey,
			hint: "h 予定  ←→ 列  ↑↓ 選択  Enter 操作  v 詳細  p プロジェクト  Tab 切替"}
	}
	return u.tabs[u.tab]
}

func (u *boardUI) switchView() {
	u.editor = inputSession{}
	u.detail.ClearSelection()
	u.footer.SwitchToPage("keys")
	tab := u.currentView()
	u.pages.SwitchToPage(tab.page)
	u.app.SetFocus(tab.body)
	u.hint.SetText(tab.hint + "  Ctrl+G 既定の担当")
	u.updateHeader()
	u.showWorkDir()
}

func (u *boardUI) updateHeader() {
	labels := make([]string, len(u.tabs))
	for i, tab := range u.tabs {
		labels[i] = tab.label
	}
	labels[u.tab] = "[black:aqua:b] " + labels[u.tab] + " [-:-:-]"
	if u.tab == automationTab {
		subview := "予定"
		if u.showHistory {
			subview = "実行履歴"
		}
		labels[u.tab] += " › " + subview
	}
	u.header.SetText(fmt.Sprintf("[white::b]Factory Board[-:-:-]   %s   [yellow]入力待ち %d件[-]", strings.Join(labels, "  "), u.agents.Blocked))
}
