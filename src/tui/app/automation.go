package app

import (
	"errors"
	"fmt"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/application/scheduler"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/schedule"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/tui/automation"
)

func (u *boardUI) newScheduleList() *automation.List {
	return automation.NewList(automation.ListActions{
		New:        u.newScheduleForm,
		Open:       func(r schedule.Rule) { u.openDetail(fmt.Sprintf("定期実行 #%d", r.ID), "", r.ID, nil) },
		EditTiming: u.editScheduleTiming,
		Remove:     func(r schedule.Rule) { u.editCommand(fmt.Sprintf("remove %d", r.ID)) },
	})
}

func (u *boardUI) newScheduleForm(kind string) {
	u.selectScheduleProject(func(dir string) {
		u.openScheduleForm(schedule.Rule{Dir: dir, Agent: u.board.state.AgentForNewItem("")}, kind)
	})
}

func (u *boardUI) openScheduleForm(r schedule.Rule, kind string) {
	closeForm := func() { u.closeScheduleForm(r.ID != 0) }
	form := automation.NewForm(r, kind, func(values automation.Settings) {
		u.saveScheduleForm(r, kind, values)
	}, closeForm, u.showWorkDir)
	u.scheduleForm = form
	u.addVoiceButton(form.Form, form.Prompt)
	u.pages.AddAndSwitchToPage("schedule-form", form, true)
	u.notice.SetText("")
	u.hint.SetText(u.voiceHint(automation.FormHint))
	u.showWorkDir()
	u.app.EnablePaste(true).SetFocus(form)
}

func (u *boardUI) saveScheduleForm(r schedule.Rule, kind string, values automation.Settings) {
	if r.ID == 0 {
		r.Agent, r.Prompt = values.Agent, values.Prompt
		id, err := scheduler.RegisterRule(u.store, u.herdr, r, kind, values.Timing)
		if id != 0 {
			u.closeScheduleForm(false)
		}
		if err != nil {
			u.notice.SetText(err.Error())
		}
		return
	}

	current, err := scheduler.ChangeSettings(u.store, r, values.Timing, values.Prompt, values.Agent)
	if err != nil {
		if current != nil {
			u.openScheduleForm(*current, current.TimingKind())
		}
		u.notice.SetText(err.Error())
		return
	}
	u.closeScheduleForm(true)
}

func (u *boardUI) closeScheduleForm(editing bool) {
	u.pages.RemovePage("schedule-form")
	u.scheduleForm = nil
	u.app.EnablePaste(false)
	u.notice.SetText("")
	if editing {
		u.pages.SwitchToPage("detail")
		u.app.SetFocus(u.detail)
	} else {
		u.switchView()
	}
	u.refresh()
}

func (u *boardUI) editScheduleTiming(r schedule.Rule) {
	label, initial := automation.TimingInput(r)
	u.openInput(label, initial, func(value string) {
		if err := scheduler.ChangeTiming(u.store, r, value); err != nil {
			var conflict *scheduler.TimingConflict
			if errors.As(err, &conflict) {
				u.editScheduleTiming(conflict.Current)
			}
			u.notice.SetText(err.Error())
			return
		}
		u.closeScheduleEditor()
		u.refresh()
	}, u.closeScheduleEditor)
	u.notice.SetText("")
}

func (u *boardUI) closeScheduleEditor() {
	u.editor = inputSession{}
	u.notice.SetText("")
	if page, _ := u.pages.GetFrontPage(); page == "detail" {
		u.footer.SwitchToPage("keys")
		u.detail.Render()
		u.app.SetFocus(u.detail)
		return
	}
	u.switchView()
}
