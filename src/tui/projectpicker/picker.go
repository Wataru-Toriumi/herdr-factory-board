// Package projectpicker は登録済みの作業先とファイルシステムからフォルダを選ぶ画面を提供する。
package projectpicker

import (
	"os"
	"path/filepath"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/tui/widgets"
	"github.com/rivo/tview"
)

// Picker は一覧の表示と選択結果をアプリへ渡す。
type Picker struct {
	Show        func(list *tview.List, path, hint string)
	Register    func(path string) error
	ReportError func(error)
}

func (p Picker) Open(projects []string, current string, filter bool, choose func(string)) {
	browse := func() { p.pickDirectory(current, choose) }
	if len(projects) == 0 {
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
	for _, dir := range projects {
		dir := dir
		list.AddItem(tview.Escape(filepath.Base(dir)), tview.Escape(dir), 0, func() { choose(dir) })
		if dir == current {
			list.SetCurrentItem(list.GetItemCount() - 1)
		}
	}
	list.AddItem("＋ フォルダを追加", "", 0, browse)
	p.Show(list, "", "↑↓ 選択  Enter 決定  Esc 戻る")
}

func (p Picker) pickDirectory(start string, choose func(string)) {
	if start != "" {
		if err := p.browseDirectory(start, choose); err == nil {
			return
		}
	}
	home, err := os.UserHomeDir()
	if err == nil {
		err = p.browseDirectory(home, choose)
	}
	if err != nil {
		p.ReportError(err)
	}
}

func (p Picker) browseDirectory(path string, choose func(string)) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	list := widgets.NewList().ShowSecondaryText(false)
	list.SetTitle(" 作業先を選択 ").SetTitleAlign(tview.AlignLeft)
	list.AddItem("このフォルダを作業先にする", "", 0, func() {
		if err := p.Register(path); err != nil {
			p.ReportError(err)
			return
		}
		choose(path)
	})

	open := func(next string) {
		if err := p.browseDirectory(next, choose); err != nil {
			p.ReportError(err)
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

	p.Show(list, path, "↑↓ 選択  Enter 開く・決定  Esc キャンセル")
	return nil
}
