package app

import "github.com/gdamore/tcell/v2"

type inputSession struct {
	submit func(string)
	cancel func()
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
	default:
		return event
	}
	return nil
}

func (u *boardUI) editCommand(initial string) {
	u.openInput("> ", initial, func(value string) {
		u.switchView()
		u.execute(value)
	}, u.switchView)
}
