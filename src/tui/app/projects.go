package app

import (
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/tui/projectpicker"
	"github.com/rivo/tview"
)

func (u *boardUI) pickProject(current string, filter bool, selected func(string)) {
	picker := projectpicker.Picker{
		Show:        u.showProjectList,
		Register:    u.store.RegisterProject,
		ReportError: func(err error) { u.notice.SetText(err.Error()) },
	}
	picker.Open(u.projects, current, filter, func(dir string) {
		u.closeDirectory()
		selected(dir)
	})
}

func (u *boardUI) showProjectList(list *tview.List, path, hint string) {
	if path != "" {
		u.context.SetText(path)
		u.notice.SetText("")
	}
	u.pages.AddAndSwitchToPage("directory", list, true)
	u.hint.SetText(hint)
	u.app.SetFocus(list)
}

func (u *boardUI) closeDirectory() {
	u.pages.RemovePage("directory")
	u.notice.SetText("")
	if u.detail.Card() != nil {
		u.pages.SwitchToPage("detail")
		u.detail.Render()
		u.app.SetFocus(u.detail)
		u.showWorkDir()
		return
	}
	u.switchView()
}

func (u *boardUI) pickCardProject(seen board.Card) {
	u.pickProject(seen.Dir, false, func(dir string) {
		err := u.store.ChangeCardProject(seen, dir)
		u.refresh()
		if err != nil {
			u.notice.SetText(err.Error())
		}
	})
}

func (u *boardUI) selectScheduleProject(selected func(string)) {
	switch {
	case u.workDir != "":
		selected(u.workDir)
	case len(u.projects) == 1:
		selected(u.projects[0])
	default:
		u.pickProject(u.workDir, false, selected)
	}
}
