package tui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/application/command"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/application/scheduler"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/schedule"
	"github.com/gdamore/tcell/v2"
)

type inputSession struct {
	submit     func(string)
	cancel     func()
	dir, agent string
}

func (u *boardUI) openInput(label, value string, submit func(string), cancel func()) {
	u.editor = inputSession{submit: submit, cancel: cancel}
	u.input.SetLabel(label).SetText(value)
	u.footer.SwitchToPage("input")
	u.app.SetFocus(u.input)
}

func (u *boardUI) inputKey(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEnter:
		u.editor.submit(u.input.GetText())
	case tcell.KeyEscape:
		u.editor.cancel()
	case tcell.KeyCtrlG:
		if u.editor.dir == "" {
			return event
		}
		u.editor.agent = nextAgent(u.editor.agent)
		u.showWorkDir()
	default:
		return event
	}
	return nil
}

func (u *boardUI) editCommand(initial string) {
	label := "> "
	switch initial {
	case "daily ":
		label = "時刻 HH:MM と作業内容 > "
	case "every ":
		label = "間隔（例 15m）と作業内容 > "
	}
	u.openInput(label, initial, func(value string) {
		dir := u.editor.dir
		cmd := command.Parse(value)
		cmd.Agent = u.editor.agent
		u.switchView()
		u.executeAt(cmd, dir)
	}, u.switchView)
}

func (u *boardUI) editForProject(line, dir string) {
	u.editCommand(line)
	u.editor.dir = dir
	u.showWorkDir()
}

func (u *boardUI) editFeedback() {
	label := "再設計のコメント > "
	if u.detailCard.CanReviewResult() {
		label = "実装の修正内容 > "
	}
	u.openInput(label, "", func(value string) {
		if strings.TrimSpace(value) == "" {
			u.notice.SetText("修正内容を入力してください")
			return
		}
		u.closeFeedbackEditor()
		u.launchDetailPhase(u.detailCard.Phase, value)
	}, func() {
		u.closeFeedbackEditor()
		u.refresh()
	})
}

func (u *boardUI) closeFeedbackEditor() {
	u.editor = inputSession{}
	u.notice.SetText("")
	u.footer.SwitchToPage("keys")
	u.app.SetFocus(u.detail)
}

func (u *boardUI) editScheduleTiming(r schedule.Rule) {
	label := "時刻（HH:MM）"
	if r.TimingKind() == schedule.Every {
		label = "間隔（例 15m）"
	}
	u.openInput(fmt.Sprintf("#%d %s > ", r.ID, label), r.TimingValue(), func(value string) {
		if err := scheduler.ChangeTiming(u.store, r, value); err != nil {
			var conflict *scheduler.TimingConflict
			if errors.As(err, &conflict) {
				u.editScheduleTiming(conflict.Current)
			}
			u.notice.SetText(err.Error())
			return
		}
		u.closeTimingEditor()
		u.refresh()
	}, u.closeTimingEditor)
	u.notice.SetText("")
}

func (u *boardUI) closeTimingEditor() {
	u.editor = inputSession{}
	u.notice.SetText("")
	if page, _ := u.pages.GetFrontPage(); page == "detail" {
		u.footer.SwitchToPage("keys")
		u.renderDetail()
		u.app.SetFocus(u.detail)
		return
	}
	u.switchView()
}
