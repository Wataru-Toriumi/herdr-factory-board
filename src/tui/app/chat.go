package app

import (
	"github.com/Wataru-Toriumi/herdr-factory-board/src/application/worker"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/tui/detail"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const chatHint = "Ctrl+S 送信   Enter 改行   PgUp/PgDn 会話のスクロール   Esc 詳細に戻る"

func (u *boardUI) openChat() {
	u.chatInput = tview.NewTextArea().SetLabel("メッセージ").SetSize(3, 0)
	u.chatForm = tview.NewForm().AddFormItem(u.chatInput)
	u.setChatButtons()
	u.chatForm.SetBorderPadding(0, 0, 0, 0)
	page := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(u.detail, 0, 1, false).
		AddItem(u.chatForm, 6, 0, true)
	u.pages.AddAndSwitchToPage("chat", page, true)
	u.showWorkDir()
	u.detail.ScrollToEnd()
	u.hint.SetText(u.voiceHint(chatHint))
	u.notice.SetText("")
	u.app.EnablePaste(true).SetFocus(u.chatForm)
}

func (u *boardUI) chatKey(e *tcell.EventKey) *tcell.EventKey {
	switch e.Key() {
	case tcell.KeyCtrlS:
		u.sendChat()
	case tcell.KeyCtrlR:
		if u.voiceHelper != "" {
			u.openVoiceInput(u.chatForm, u.chatInput)
		}
	case tcell.KeyPgUp, tcell.KeyPgDn:
		u.detail.InputHandler()(e, nil)
	case tcell.KeyEscape:
		u.pages.RemovePage("chat")
		u.chatForm, u.chatInput = nil, nil
		u.pages.SwitchToPage("detail")
		u.app.EnablePaste(false).SetFocus(u.detail)
		u.refresh()
		u.detail.Render()
	default:
		return e
	}
	return nil
}

func (u *boardUI) sendChat() {
	err := worker.SendChat(u.store, *u.detail.Card(), u.chatInput.GetText())
	u.refresh()
	if err != nil {
		u.notice.SetText(err.Error())
		return
	}
	u.chatInput.SetText("", true)
	u.chatForm.SetFocus(0)
	u.app.SetFocus(u.chatForm)
	u.notice.SetText("")
	u.detail.ScrollToEnd()
}

func (u *boardUI) refreshChat(b board.State) {
	before := u.detail.Card()
	u.detail.Refresh(b)
	after := u.detail.Card()
	if before != after {
		if before.CanApplyChat() != after.CanApplyChat() || before.CanReviewDesign() != after.CanReviewDesign() || before.CanReviewResult() != after.CanReviewResult() {
			u.setChatButtons()
		}
		u.detail.ScrollToEnd()
	}
	u.hint.SetText(u.voiceHint(chatHint))
}

// 詳細画面と同じ適用・承認操作を、入力中はフォームのボタンから選べるようにする。
func (u *boardUI) setChatButtons() {
	_, focusedButton := u.chatForm.GetFocusedItemIndex()
	u.chatForm.ClearButtons().AddButton("送信", u.sendChat)
	u.addVoiceButton(u.chatForm, u.chatInput)
	for _, shortcut := range detail.Shortcuts(u.detail.Card(), false) {
		action := shortcut.Action
		switch action {
		case detail.ApplyChat, detail.StartImplementation, detail.ApproveResult:
			u.chatForm.AddButton(shortcut.Label, func() { u.runDetailAction(action) })
		}
	}
	if focusedButton >= 0 {
		u.chatForm.SetFocus(0)
		u.app.SetFocus(u.chatForm)
	}
}
