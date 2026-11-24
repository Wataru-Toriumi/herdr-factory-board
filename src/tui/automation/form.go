package automation

import (
	"fmt"
	"strings"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/schedule"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const FormHint = "Tab 項目移動   Enter 本文で改行   Ctrl+S 保存   Esc 取消"

type Settings struct {
	Timing, Prompt, Agent string
}

type Form struct {
	*tview.Form
	dir, agent                 string
	save, cancel, agentChanged func()
}

func NewForm(r schedule.Rule, kind string, submit func(Settings), cancel, agentChanged func()) *Form {
	label, example := "毎日の時刻", "09:00"
	if kind == schedule.Every {
		label, example = "実行間隔", "15m"
	}
	timing := tview.NewInputField().SetLabel(label).SetPlaceholder(example).SetFieldWidth(12).SetText(r.TimingValue())
	prompt := tview.NewTextArea().SetLabel("実行内容").SetSize(6, 0).SetText(r.Prompt, true)
	f := &Form{
		Form:         tview.NewForm(),
		dir:          r.Dir,
		agent:        r.Agent,
		cancel:       cancel,
		agentChanged: agentChanged,
	}
	f.save = func() {
		submit(Settings{Timing: strings.TrimSpace(timing.GetText()), Prompt: prompt.GetText(), Agent: f.agent})
	}
	f.AddFormItem(timing).AddFormItem(prompt).AddButton("保存", f.save).AddButton("取消", cancel)
	title := " 定期実行の登録 "
	if r.ID != 0 {
		title = fmt.Sprintf(" 定期実行 #%d の編集 ", r.ID)
		f.SetFocus(1)
	}
	f.SetBorder(true).SetTitle(title)
	return f
}

func (f *Form) Context() string {
	return "担当: " + board.AgentLabel(f.agent) + "（Ctrl+G 切替）  作業先: " + f.dir
}

func (f *Form) HandleKey(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyCtrlS:
		f.save()
	case tcell.KeyEscape:
		f.cancel()
	case tcell.KeyCtrlG:
		f.agent = board.NextAgent(f.agent)
		f.agentChanged()
	default:
		return event
	}
	return nil
}
