package app

import (
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/dictation"
	"github.com/rivo/tview"
)

type voiceDialog struct {
	session      *dictation.Session
	dialog       *tview.Modal
	form         *tview.Form
	prompt       *tview.TextArea
	ready        bool
	previousHint string
}

func (u *boardUI) voiceHint(hint string) string {
	if u.voiceHelper == "" {
		return hint
	}
	return "Ctrl+R 音声入力  " + hint
}

func (u *boardUI) addVoiceButton(form *tview.Form, prompt *tview.TextArea) {
	if u.voiceHelper != "" {
		form.AddButton("音声入力", func() { u.openVoiceInput(form, prompt) })
	}
}

func (u *boardUI) openVoiceInput(form *tview.Form, prompt *tview.TextArea) {
	session, err := dictation.Start(u.voiceHelper)
	if err != nil {
		u.notice.SetText(err.Error())
		return
	}
	dialog := tview.NewModal().SetText("マイクを準備中です。初回はmacOSの許可画面を確認してください。").
		AddButtons([]string{"停止", "取消"}).
		SetDoneFunc(func(index int, _ string) {
			if index == 0 {
				u.stopVoiceInput()
			} else {
				u.finishVoiceInput("", nil)
			}
		})
	u.voice = &voiceDialog{session: session, form: form, prompt: prompt, dialog: dialog, previousHint: u.hint.GetText(false)}
	u.pages.AddPage("dictation", dialog, true, true)
	u.notice.SetText("")
	u.hint.SetText("Ctrl+R 録音停止  Esc 取消")
	u.app.SetFocus(dialog)
}

func (u *boardUI) stopVoiceInput() {
	v := u.voice
	if !v.ready {
		return
	}
	if err := v.session.Stop(); err != nil {
		// ヘルパーが先に終了したときは、その認識結果または診断を受け取る。
		u.updateVoiceInput()
		return
	}
	v.ready = false
	v.dialog.SetText("文字起こし中…")
}

// 画面更新時に通知を受け取り、フォームの操作と同じ goroutine で本文へ反映する。
func (u *boardUI) updateVoiceInput() {
	v := u.voice
	if v == nil {
		return
	}
	for {
		select {
		case event, ok := <-v.session.Updates:
			if !ok {
				u.finishVoiceInput("", nil)
				return
			}
			if event.Ready {
				v.ready = true
				v.dialog.SetText("録音中です。話し終えたら「停止」を押してください。")
				continue
			}
			u.finishVoiceInput(event.Text, event.Err)
			return
		default:
			return
		}
	}
}

func (u *boardUI) finishVoiceInput(text string, err error) {
	v := u.voice
	v.session.Cancel()
	u.voice = nil
	u.pages.RemovePage("dictation")
	if err != nil {
		u.notice.SetText(err.Error())
	} else if text != "" {
		_, start, end := v.prompt.GetSelection()
		v.prompt.Replace(start, end, text)
		u.notice.SetText("文字起こしを挿入しました")
	}
	v.form.SetFocus(v.form.GetFormItemIndex(v.prompt.GetLabel()))
	u.app.SetFocus(v.form)
	u.hint.SetText(v.previousHint)
	u.showWorkDir()
}
