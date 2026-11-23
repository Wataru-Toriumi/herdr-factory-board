package tui

import (
	"os"
	"path/filepath"

	"github.com/rivo/tview"
)

func (u *boardUI) pickDirectory(start string, choose func(string)) {
	if start != "" {
		if err := u.browseDirectory(start, choose); err == nil {
			return
		}
	}
	home, err := os.UserHomeDir()
	if err == nil {
		err = u.browseDirectory(home, choose)
	}
	if err != nil {
		u.notice.SetText(err.Error())
	}
}

func (u *boardUI) closeDirectory() {
	u.pages.RemovePage("directory")
	u.notice.SetText("")
	if u.detailCard != nil {
		u.pages.SwitchToPage("detail")
		u.renderCardDetail(*u.detailCard)
		u.app.SetFocus(u.detail)
		u.showWorkDir()
		return
	}
	u.switchView()
}

func (u *boardUI) browseDirectory(path string, choose func(string)) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	list := boardList().ShowSecondaryText(false)
	list.SetTitle(" 作業先を選択 ").SetTitleAlign(tview.AlignLeft)
	list.AddItem("このフォルダを作業先にする", "", 0, func() {
		if err := u.store.RegisterProject(path); err != nil {
			u.notice.SetText(err.Error())
			return
		}
		choose(path)
	})

	open := func(next string) {
		if err := u.browseDirectory(next, choose); err != nil {
			u.notice.SetText(err.Error())
		}
	}
	if parent := filepath.Dir(path); parent != path {
		list.AddItem("..  親フォルダへ", "", 0, func() { open(parent) })
	}
	for _, entry := range entries {
		next := filepath.Join(path, entry.Name())
		if !entry.IsDir() {
			if entry.Type()&os.ModeSymlink == 0 {
				continue
			}
			info, err := os.Stat(next)
			if err != nil || !info.IsDir() {
				continue
			}
		}
		list.AddItem(tview.Escape(entry.Name())+"/", "", 0, func() { open(next) })
	}

	u.context.SetText(path)
	u.notice.SetText("")
	u.hint.SetText("↑↓ 選択  Enter 開く・決定  Esc キャンセル")
	u.pages.AddAndSwitchToPage("directory", list, true)
	u.app.SetFocus(list)
	return nil
}
