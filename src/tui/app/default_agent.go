package app

import (
	"slices"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/rivo/tview"
)

func (u *boardUI) pickDefaultAgent() {
	agents := board.Agents()
	labels := make([]string, len(agents))
	for i, agent := range agents {
		labels[i] = board.AgentLabel(agent)
	}
	dialog := tview.NewModal().
		SetText("このワークスペースで新しいタスク・予定に使う既定の担当").
		AddButtons(labels).
		SetFocus(slices.Index(agents, u.board.state.AgentForNewItem("")))
	dialog.SetDoneFunc(func(index int, _ string) {
		u.pages.RemovePage("default-agent")
		u.switchView()
		if index < 0 {
			return
		}
		agent := agents[index]
		err := u.store.Edit(func(b *board.State) error {
			b.DefaultAgent = agent
			return nil
		})
		if err != nil {
			u.notice.SetText(err.Error())
			return
		}
		u.refresh()
		u.notice.SetText("既定の担当を " + board.AgentLabel(agent) + " に設定しました")
	})
	u.pages.AddPage("default-agent", dialog, true, true)
	u.notice.SetText("")
	u.hint.SetText("←→ 担当を選択   Enter 保存   Esc 取消")
	u.app.SetFocus(dialog)
}
