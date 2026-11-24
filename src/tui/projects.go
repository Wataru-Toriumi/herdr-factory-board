package tui

import (
	"path/filepath"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/tui/widgets"
	"github.com/rivo/tview"
)

func (u *boardUI) pickProject(current string, filter bool, selected func(string)) {
	choose := func(dir string) {
		u.closeDirectory()
		selected(dir)
	}
	browse := func() { u.pickDirectory(current, choose) }
	if len(u.projects) == 0 {
		browse()
		return
	}
	list := widgets.NewList()
	title := " 作業先を選択 "
	if filter {
		title = " 表示するプロジェクト "
		list.AddItem("すべて", "", 0, func() { choose("") })
	}
	list.SetTitle(title).SetTitleAlign(tview.AlignLeft)
	for _, dir := range u.projects {
		dir := dir
		list.AddItem(tview.Escape(filepath.Base(dir)), tview.Escape(dir), 0, func() { choose(dir) })
		if dir == current {
			list.SetCurrentItem(list.GetItemCount() - 1)
		}
	}
	list.AddItem("＋ フォルダを追加", "", 0, browse)
	u.pages.AddAndSwitchToPage("directory", list, true)
	u.hint.SetText("↑↓ 選択  Enter 決定  Esc 戻る")
	u.app.SetFocus(list)
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
