package app

import (
	"github.com/Wataru-Toriumi/herdr-factory-board/src/application/command"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/tui/taskboard"
)

func (u *boardUI) newTaskForm() {
	dir := u.workDir
	if dir == "" && len(u.projects) == 1 {
		dir = u.projects[0]
	}
	u.taskForm = taskboard.NewForm(dir, u.saveTaskForm, u.pickTaskProject, func() {
		u.notice.SetText("")
		u.closeTaskForm()
	})
	u.pages.AddAndSwitchToPage("task-form", u.taskForm, true)
	u.notice.SetText("")
	u.focusTaskForm()
}

func (u *boardUI) focusTaskForm() {
	u.hint.SetText(taskboard.FormHint)
	u.showWorkDir()
	u.app.EnablePaste(true).SetFocus(u.taskForm)
}

func (u *boardUI) pickTaskProject() {
	u.pickProject(u.taskForm.Project, false, func(dir string) {
		u.taskForm.Project = dir
		u.refresh()
	})
}

func (u *boardUI) saveTaskForm(prompt, agent string) {
	dir := u.taskForm.Project
	if dir == "" {
		u.pickTaskProject()
		return
	}
	u.notice.SetText("")
	if err := u.runCommand(command.Command{Name: "add", Arg: prompt, Agent: agent}, dir); err == nil {
		u.closeTaskForm()
	}
}

func (u *boardUI) closeTaskForm() {
	u.pages.RemovePage("task-form")
	u.taskForm = nil
	u.app.EnablePaste(false)
	u.switchView()
}
