package main

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
)

func (u *boardUI) read(title, text string, scheduleID int) {
	u.detailBody, u.detailScheduleID = text, scheduleID
	u.detail.SetTitle(" " + title + " ")
	u.detail.SetText(text).ScrollToBeginning()
	u.pages.SwitchToPage("detail")
	u.hint.SetText("↑↓ スクロール   Esc 戻る")
	u.updateDetailTiming()
	u.app.SetFocus(u.detail)
}

func (u *boardUI) updateDetailTiming() {
	if u.detailScheduleID == 0 {
		return
	}
	timing := "解除済み"
	hint := "↑↓ スクロール   Esc 戻る"
	if r, ok := u.schedules.byID(u.detailScheduleID); ok {
		timing = r.frequency()
		hint = "t 定期実行の設定変更   ↑↓ スクロール   Esc 戻る"
	}
	text := fmt.Sprintf("定期実行 #%d  %s\n\n%s", u.detailScheduleID, timing, u.detailBody)
	if u.detail.GetText(false) != text {
		u.detail.SetText(text)
	}
	u.hint.SetText(hint)
}

func (u *boardUI) closeTimingEditor() {
	u.editingScheduleID = 0
	u.notice.SetText("")
	if page, _ := u.pages.GetFrontPage(); page == "detail" {
		u.footer.SwitchToPage("keys")
		u.updateDetailTiming()
		u.app.SetFocus(u.detail)
		return
	}
	u.switchView()
}

func (u *boardUI) help() {
	var text strings.Builder
	fmt.Fprintln(&text, "タスクを選んで v で詳細を表示。Enter で開始・担当ペインへ移動・結果表示。")
	fmt.Fprintln(&text, "p で作業先を選択、n でタスクを追加。初回はフォルダ選択から始めます。")
	fmt.Fprintln(&text, "コマンドはそのまま入力し、Enter で確定。Esc でキャンセル。")
	fmt.Fprintln(&text, "dir 絶対パス / add 作業内容 / daily HH:MM 作業内容")
	fmt.Fprintln(&text, "start 番号 / open 番号 / result 番号 / recover 番号")
	fmt.Fprintln(&text, "scheduler / remove 定期番号")
	fmt.Fprintln(&text, "自動化タブでは n で毎日、i で間隔指定（例: every 15m 作業内容）。t で時刻・間隔の変更、Enter で詳細、x で解除。")
	fmt.Fprintln(&text, "\nワークスペース")
	for _, w := range u.spaces {
		label := w.Label
		if label == "" {
			label = w.ID
		}
		fmt.Fprintf(&text, "%s  %s\n", label, w.AgentStatus)
	}
	u.read("操作方法", text.String(), 0)
}

func (u *boardUI) edit(initial string) {
	u.input.SetLabel("> ")
	switch initial {
	case "daily ":
		u.input.SetLabel("時刻 HH:MM と作業内容 > ")
	case "every ":
		u.input.SetLabel("間隔（例 15m）と作業内容 > ")
	}
	u.input.SetText(initial)
	u.footer.SwitchToPage("input")
	u.app.SetFocus(u.input)
}

func (u *boardUI) editScheduleTiming(r schedule) {
	value, label := r.Time, "時刻（HH:MM）"
	if r.Interval != nil {
		value, label = fmt.Sprintf("%dm", r.Interval.Minutes), "間隔（例 15m）"
	}
	u.edit(value)
	u.input.SetLabel(fmt.Sprintf("#%d %s > ", r.ID, label))
	u.editingScheduleID = r.ID
	u.notice.SetText("")
}

func (u *boardUI) submitInput() {
	value := u.input.GetText()
	if u.editingScheduleID == 0 {
		u.switchView()
		u.execute(value)
		return
	}
	if err := updateScheduleTiming(u.store, u.editingScheduleID, value); err != nil {
		u.notice.SetText(err.Error())
		return
	}
	u.closeTimingEditor()
	u.refresh()
}

func (u *boardUI) composeAtWorkDir(prefix string) {
	if u.workDir == "" {
		u.pickDirectory(prefix)
	} else {
		u.edit(prefix)
	}
}

func (u *boardUI) execute(line string) {
	u.notice.SetText("")
	c, err := command(u.store, u.herdr, line)
	u.refresh()
	if err != nil {
		u.notice.SetText(err.Error())
	} else if c != nil {
		u.read("カードの詳細", fmt.Sprintf("#%d  %s  %s\n%s\n%s", c.ID, c.Status, c.Dir, c.Prompt, c.Result), c.sourceScheduleID())
	}
}

func (u *boardUI) activate() {
	switch u.tab {
	case automationTab:
		if r, ok := u.schedules.selected(); ok {
			u.read(fmt.Sprintf("定期実行 #%d", r.ID), fmt.Sprintf("作業先: %s\n\n%s", r.Dir, r.Prompt), r.ID)
		}
		return
	case agentTab:
		i := u.agents.GetCurrentItem()
		if i >= 0 && i < len(u.rows) {
			if err := u.rows[i].focus(u.store, u.herdr); err != nil {
				u.notice.SetText(err.Error())
			}
		}
		return
	}
	c, ok := u.selectedCard()
	if !ok {
		return
	}
	action := "result"
	switch c.Status {
	case todo:
		action = "start"
	case running:
		action = "open"
	}
	u.execute(fmt.Sprintf("%s %d", action, c.ID))
}

func (u *boardUI) key(event *tcell.EventKey) *tcell.EventKey {
	key := event.Key()
	switch u.activePage() {
	case "directory":
		if key == tcell.KeyEscape {
			u.closeDirectory()
			return nil
		}
		return event
	case "input":
		switch key {
		case tcell.KeyEscape:
			if u.editingScheduleID != 0 {
				u.closeTimingEditor()
			} else {
				u.switchView()
			}
			return nil
		case tcell.KeyEnter:
			u.submitInput()
			return nil
		}
		return event
	case "detail":
		if key == tcell.KeyRune && event.Rune() == 't' {
			if r, ok := u.schedules.byID(u.detailScheduleID); ok {
				u.editScheduleTiming(r)
			}
			return nil
		}
		if key == tcell.KeyEscape {
			u.switchView()
			return nil
		}
		return event
	}
	switch key {
	case tcell.KeyTab, tcell.KeyBacktab:
		step := 1
		if key == tcell.KeyBacktab {
			step = -1
		}
		u.tab = (u.tab + step + len(tabLabels)) % len(tabLabels)
		u.notice.SetText("")
		u.switchView()
	case tcell.KeyLeft, tcell.KeyRight:
		if u.tab == taskTab {
			direction := 1
			if key == tcell.KeyLeft {
				direction = -1
			}
			u.lane = (u.lane + direction + len(laneNames)) % len(laneNames)
			u.layoutColumns()
			u.app.SetFocus(u.lists[u.lane])
			u.highlightLane()
		}
	case tcell.KeyEnter:
		u.activate()
	case tcell.KeyRune:
		if u.tab == taskTab && event.Rune() == 'v' {
			if c, ok := u.selectedCard(); ok {
				u.execute(fmt.Sprintf("result %d", c.ID))
			}
			return nil
		}
		if u.tab == automationTab {
			switch event.Rune() {
			case 't':
				if r, ok := u.schedules.selected(); ok {
					u.editScheduleTiming(r)
				}
				return nil
			case 'x':
				if r, ok := u.schedules.selected(); ok {
					u.edit(fmt.Sprintf("remove %d", r.ID))
				}
				return nil
			}
		}
		switch event.Rune() {
		case 'n':
			if u.tab == automationTab {
				u.composeAtWorkDir("daily ")
			} else {
				u.composeAtWorkDir("add ")
			}
		case 'i':
			if u.tab == automationTab {
				u.composeAtWorkDir("every ")
			} else {
				u.edit("i")
			}
		case 'p':
			u.pickDirectory("")
		case 'q', 'Q':
			u.app.Stop()
		case ':':
			u.edit("")
		case '?':
			u.help()
		default:
			u.edit(string(event.Rune()))
		}
	default:
		return event
	}
	return nil
}
