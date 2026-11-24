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
			hint: automation.ListHint},
	}
	for i, tab := range u.tabs {
		u.pages.AddPage(tab.page, tab.body, true, i == u.tab)
	}
}

func (u *boardUI) switchView() {
	u.editor = inputSession{}
	u.detail.ClearSelection()
	u.footer.SwitchToPage("keys")
	tab := u.tabs[u.tab]
	u.pages.SwitchToPage(tab.page)
	focus := tab.body
	if u.tab == taskTab {
		focus = u.tasks.List(u.tasks.Lane())
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
	u.header.SetText(fmt.Sprintf("[white::b]Factory Board[-:-:-]   %s   [yellow]入力待ち %d件[-]", strings.Join(labels, "  "), u.agents.Blocked))
}
