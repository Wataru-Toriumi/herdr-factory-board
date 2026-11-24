package tui

import (
	"fmt"
	"strings"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/application/scheduler"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/schedule"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type scheduleForm struct {
	rule        schedule.Rule
	kind, agent string
	timing      *tview.InputField
	prompt      *tview.TextArea
}

func (u *boardUI) newScheduleForm(kind string) {
	u.selectScheduleProject(func(dir string) { u.openScheduleForm(schedule.Rule{Dir: dir}, kind) })
}

func (u *boardUI) openScheduleForm(r schedule.Rule, kind string) {
	label, example := "毎日の時刻", "09:00"
	if kind == schedule.Every {
		label, example = "実行間隔", "15m"
	}
	f := &scheduleForm{
		rule: r, kind: kind, agent: r.Agent,
		timing: tview.NewInputField().SetLabel(label).SetPlaceholder(example).SetFieldWidth(12).SetText(r.TimingValue()),
		prompt: tview.NewTextArea().SetLabel("実行内容").SetSize(6, 0).SetText(r.Prompt, true),
	}
	u.scheduleForm = f
	form := tview.NewForm().AddFormItem(f.timing).AddFormItem(f.prompt).
		AddButton("保存", u.saveScheduleForm).AddButton("取消", u.closeScheduleForm)
	title := " 定期実行の登録 "
	if r.ID != 0 {
		title = fmt.Sprintf(" 定期実行 #%d の編集 ", r.ID)
		form.SetFocus(1)
	}
	form.SetBorder(true).SetTitle(title)
	u.pages.AddAndSwitchToPage("schedule-form", form, true)
	u.notice.SetText("")
	u.hint.SetText("Tab 項目移動   Enter 本文で改行   Ctrl+S 保存   Esc 取消")
	u.showWorkDir()
	u.app.EnablePaste(true).SetFocus(form)
}

func (u *boardUI) scheduleFormKey(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyCtrlS:
		u.saveScheduleForm()
	case tcell.KeyEscape:
		u.closeScheduleForm()
	case tcell.KeyCtrlG:
		u.scheduleForm.agent = nextAgent(u.scheduleForm.agent)
		u.showWorkDir()
	default:
		return event
	}
	return nil
}

func (u *boardUI) saveScheduleForm() {
	f := u.scheduleForm
	timing := strings.TrimSpace(f.timing.GetText())
	prompt := f.prompt.GetText()
	if f.rule.ID == 0 {
		r := f.rule
		r.Agent, r.Prompt = f.agent, prompt
		id, err := scheduler.RegisterRule(u.store, u.herdr, r, f.kind, timing)
		if id != 0 {
			u.closeScheduleForm()
		}
		if err != nil {
			u.notice.SetText(err.Error())
		}
		return
	}

	current, err := scheduler.ChangeSettings(u.store, f.rule, timing, prompt, f.agent)
	if err != nil {
		if current != nil {
			u.openScheduleForm(*current, current.TimingKind())
		}
		u.notice.SetText(err.Error())
		return
	}
	u.closeScheduleForm()
}

func (u *boardUI) closeScheduleForm() {
	editing := u.scheduleForm.rule.ID != 0
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
