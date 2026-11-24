package tui

import "github.com/gdamore/tcell/v2"

func (u *boardUI) key(event *tcell.EventKey) *tcell.EventKey {
	key := event.Key()
	switch u.activePage() {
	case "directory":
		if key == tcell.KeyEscape {
			u.closeDirectory()
			u.refresh()
			return nil
		}
		return event
	case "schedule-form":
		return u.scheduleFormKey(event)
	case "input":
		return u.inputKey(event)
	case "detail":
		if key == tcell.KeyRune && u.detailKey(event.Rune()) {
			return nil
		}
		if key == tcell.KeyEscape {
			u.switchView()
			return nil
		}
		return event
	}
	if u.tabs[u.tab].handleKey(event) {
		return nil
	}
	switch key {
	case tcell.KeyTab, tcell.KeyBacktab:
		step := 1
		if key == tcell.KeyBacktab {
			step = -1
		}
		u.tab = (u.tab + step + len(u.tabs)) % len(u.tabs)
		u.notice.SetText("")
		u.switchView()
	case tcell.KeyLeft, tcell.KeyRight:
		// レーンを持たない一覧でも左右キーはここで消費する。
		return nil
	case tcell.KeyRune:
		switch event.Rune() {
		case 'n':
			u.pickProject(u.workDir, false, func(dir string) { u.editForProject("add ", dir) })
		case 'p':
			u.pickProject(u.workDir, true, func(dir string) {
				u.workDir = dir
				u.refresh()
			})
		case 'q', 'Q':
			u.app.Stop()
		case ':':
			u.editCommand("")
		case '?':
			u.help()
		default:
			u.editCommand(string(event.Rune()))
		}
	default:
		return event
	}
	return nil
}
