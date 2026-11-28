package detail

import (
	"slices"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/rivo/tview"
)

const ChatHint = "Ctrl+S 送信   Enter 改行   PgUp/PgDn 会話のスクロール   Esc 詳細に戻る"

type ChatForm struct {
	*tview.Flex
	Form    *tview.Form
	Message *tview.TextArea

	view    *View
	actions ShortcutList
	run     func(Action)
}

func NewChatForm(view *View, send func(), addVoiceButton func(*tview.Form, *tview.TextArea), run func(Action)) *ChatForm {
	message := tview.NewTextArea().SetLabel("メッセージ").SetSize(3, 0)
	form := tview.NewForm().AddFormItem(message).AddButton("送信", send)
	addVoiceButton(form, message)
	form.SetBorderPadding(0, 0, 0, 0)
	page := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(view, 0, 1, false).
		AddItem(form, 6, 0, true)
	f := &ChatForm{Flex: page, Form: form, Message: message, view: view, run: run}
	f.updateActions()
	return f
}

func (f *ChatForm) Refresh(b board.State, setFocus func(tview.Primitive)) {
	before := f.view.Card()
	f.view.Refresh(b)
	if before == f.view.Card() {
		return
	}
	_, focusedButton := f.Form.GetFocusedItemIndex()
	if f.updateActions() && focusedButton >= 0 {
		f.Form.SetFocus(0)
		setFocus(f.Form)
	}
	f.view.ScrollToEnd()
}

// 入力欄と送信・音声入力ボタンを保持し、カードに応じた操作だけを差し替える。
func (f *ChatForm) updateActions() bool {
	var actions ShortcutList
	for _, shortcut := range cardShortcuts(f.view.Card()) {
		switch shortcut.Action {
		case ApplyChat, StartImplementation, ApproveResult:
			actions = append(actions, shortcut)
		}
	}
	if slices.Equal(actions, f.actions) {
		return false
	}
	for range f.actions {
		f.Form.RemoveButton(f.Form.GetButtonCount() - 1)
	}
	for _, shortcut := range actions {
		action := shortcut.Action
		f.Form.AddButton(shortcut.Label, func() { f.run(action) })
	}
	f.actions = actions
	return true
}
