package taskboard

import (
	"slices"
	"strings"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const FormHint = "Enter 改行  Tab 項目移動  Ctrl+S 保存  Esc 取消  Ctrl+G 担当切替"

type Form struct {
	*tview.Form
	Prompt                    *tview.TextArea
	Project                   string
	agent                     *tview.DropDown
	save, pickProject, cancel func()
}

func NewForm(project string, submit func(prompt, agent string), pickProject, cancel func()) *Form {
	prompt := tview.NewTextArea().SetLabel("作業内容").SetSize(6, 0)
	agent := tview.NewDropDown().SetLabel("担当")
	agents := board.Agents()
	for _, id := range agents {
		agent.AddOption(board.AgentLabel(id), nil)
	}
	agent.SetCurrentOption(0)
	f := &Form{Form: tview.NewForm(), Prompt: prompt, Project: project, agent: agent, pickProject: pickProject, cancel: cancel}
	f.save = func() {
		index, _ := agent.GetCurrentOption()
		submit(strings.TrimSpace(prompt.GetText()), agents[index])
	}
	f.AddFormItem(prompt).AddFormItem(agent).
		AddButton("保存", f.save).AddButton("作業先を選択", pickProject).AddButton("取消", cancel)
	f.SetBorder(true).SetTitle(" 新規タスク ")
	return f
}

func (f *Form) SelectAgent(agent string) {
	f.agent.SetCurrentOption(max(0, slices.Index(board.Agents(), agent)))
}

func (f *Form) Context() string {
	dir := f.Project
	if dir == "" {
		dir = "未選択"
	}
	return "作業先（Ctrl+P で選択）: " + dir
}

func (f *Form) HandleKey(event *tcell.EventKey) *tcell.EventKey {
	if f.agent.IsOpen() {
		return event
	}
	switch event.Key() {
	case tcell.KeyCtrlS:
		f.save()
	case tcell.KeyEscape:
		f.cancel()
	case tcell.KeyCtrlP:
		f.pickProject()
	case tcell.KeyCtrlG:
		index, _ := f.agent.GetCurrentOption()
		f.agent.SetCurrentOption((index + 1) % f.agent.GetOptionCount())
	default:
		return event
	}
	return nil
}
