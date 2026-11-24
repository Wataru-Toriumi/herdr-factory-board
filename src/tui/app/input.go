package app

import (
	"github.com/Wataru-Toriumi/herdr-factory-board/src/application/command"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
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
		u.editor.agent = board.NextAgent(u.editor.agent)
		u.showWorkDir()
	default:
		return event
	}
	return nil
}

func (u *boardUI) editCommand(initial string) {
	u.openInput("> ", initial, func(value string) {
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
