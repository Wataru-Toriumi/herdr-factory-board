package main

import (
	"errors"
	"os"
	"path/filepath"
	"slices"

	"github.com/rivo/tview"
)

func registerProject(s store, path string) error {
	if !filepath.IsAbs(path) {
		return errors.New("作業先の絶対パスを指定してください")
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("作業先はディレクトリを指定してください")
	}
	path = filepath.Clean(path)
	return s.edit(func(b *board) error {
		if !slices.Contains(b.Projects, path) {
			b.Projects = append(b.Projects, path)
		}
		return nil
	})
}

// 旧版の作業先と履歴もプロジェクトとして扱う。カードや予定の書き換えは不要。
func (b board) projectDirs() []string {
	seen := make(map[string]bool)
	add := func(dir string) {
		if dir != "" {
			seen[dir] = true
		}
	}
	add(b.Dir)
	for _, dir := range b.Projects {
		add(dir)
	}
	for _, c := range b.Cards {
		add(c.Dir)
	}
	for _, r := range b.Schedules {
		add(r.Dir)
	}
	dirs := make([]string, 0, len(seen))
	for dir := range seen {
		dirs = append(dirs, dir)
	}
	slices.Sort(dirs)
	return dirs
}

func projectNames(dirs []string) map[string]string {
	counts := make(map[string]int)
	for _, dir := range dirs {
		counts[filepath.Base(dir)]++
	}
	names := make(map[string]string, len(dirs))
	for _, dir := range dirs {
		name := filepath.Base(dir)
		if counts[name] > 1 {
			name = dir
		}
		names[dir] = name
	}
	return names
}

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
	list := boardList()
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

func (c card) canChangeProject() bool {
	return c.Status == todo && c.ScheduleID == 0 && c.Source == ""
}

func (u *boardUI) pickCardProject(seen card) {
	u.pickProject(seen.Dir, false, func(dir string) {
		err := u.store.editCard(seen.ID, func(c *card) error {
			if !c.canChangeProject() || c.Dir != seen.Dir {
				return errors.New("チケットが更新されました。最新の作業先と状態を確認してください")
			}
			c.Dir = dir
			return nil
		})
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

func (u *boardUI) editForProject(line, dir string) {
	u.edit(line)
	u.inputDir = dir
	u.showWorkDir()
}
