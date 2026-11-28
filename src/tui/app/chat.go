package app

import (
	"github.com/Wataru-Toriumi/herdr-factory-board/src/application/worker"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/tui/detail"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (u *boardUI) openChat() {
	u.chatForm = detail.NewChatForm(u.detail, u.sendChat, u.addVoiceButton, u.runDetailAction)
	u.pages.AddAndSwitchToPage("chat", u.chatForm, true)
	u.showWorkDir()
	u.detail.ScrollToEnd()
	u.hint.SetText(u.voiceHint(detail.ChatHint))
	u.notice.SetText("")
	u.app.EnablePaste(true).SetFocus(u.chatForm.Form)
}

func (u *boardUI) chatKey(e *tcell.EventKey) *tcell.EventKey {
	switch e.Key() {
	case tcell.KeyCtrlS:
		u.sendChat()
	case tcell.KeyCtrlR:
		if u.voiceHelper != "" {
			u.openVoiceInput(u.chatForm.Form, u.chatForm.Message)
		}
	case tcell.KeyPgUp, tcell.KeyPgDn:
		u.detail.InputHandler()(e, nil)
	case tcell.KeyEscape:
		u.pages.RemovePage("chat")
		u.chatForm = nil
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
	err := worker.SendChat(u.store, *u.detail.Card(), u.chatForm.Message.GetText())
	u.refresh()
	if err != nil {
		u.notice.SetText(err.Error())
		return
	}
	u.chatForm.Message.SetText("", true)
	u.chatForm.Form.SetFocus(0)
	u.app.SetFocus(u.chatForm.Form)
	u.notice.SetText("")
	u.detail.ScrollToEnd()
}

func (u *boardUI) refreshChat(b board.State) {
	u.chatForm.Refresh(b, func(p tview.Primitive) { u.app.SetFocus(p) })
	u.hint.SetText(u.voiceHint(detail.ChatHint))
}
