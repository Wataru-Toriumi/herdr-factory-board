package main

import (
	"os"
	"path/filepath"

	"github.com/rivo/tview"
)

func (u *boardUI) pickDirectory(nextCommand string) {
	start := u.workDir
	if start != "" {
		if err := u.browseDirectory(start, nextCommand); err == nil {
			return
		}
	}
	home, err := os.UserHomeDir()
	if err == nil {
		err = u.browseDirectory(home, nextCommand)
	}
	if err != nil {
		u.notice.SetText(err.Error())
	}
}

func (u *boardUI) closeDirectory() {
	u.pages.RemovePage("directory")
	u.notice.SetText("")
	u.showWorkDir()
	u.switchView()
}

func (u *boardUI) browseDirectory(path string, nextCommand string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	list := boardList().ShowSecondaryText(false)
	list.SetTitle(" 作業先を選択 ").SetTitleAlign(tview.AlignLeft)
	list.AddItem("このフォルダを作業先にする", "", 0, func() {
		if err := setWorkDir(u.store, path); err != nil {
			u.notice.SetText(err.Error())
			return
		}
		u.workDir = path
		u.closeDirectory()
		if nextCommand != "" {
			u.edit(nextCommand)
		}
	})

	open := func(next string) {
		if err := u.browseDirectory(next, nextCommand); err != nil {
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
