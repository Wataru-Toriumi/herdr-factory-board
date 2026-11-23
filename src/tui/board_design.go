package tui

import (
	"fmt"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/worker"
)

func (u *boardUI) renderCardDetail(c board.Card) {
	u.detailCard = &c
	body := fmt.Sprintf("#%d  %s\n作業先: %s\n%s", c.ID, c.Status, c.Dir, c.Prompt)
	if phase := c.PhaseLabel(); phase != "" {
		body += "\n\n" + phase
	}
	if c.Design != "" {
		body += "\n\n設計案\n" + c.Design
	}
	if c.Feedback != "" {
		body += "\n\nレビューコメント\n" + c.Feedback
	}
	if c.Phase != "" && c.Result != "" {
		body += "\n\n実行結果\n" + c.Result
	} else {
		body += "\n" + c.Result
	}
	u.detailBody = body
	if u.detailScheduleID != 0 {
		u.updateDetailTiming()
		return
	}
	u.detail.SetText(body)
	hint := ""
	if c.CanChangeProject() {
		hint = "p 作業先を変更   "
	}
	for _, shortcut := range u.designShortcuts() {
		hint += string(shortcut.key) + " " + shortcut.label + "   "
	}
	u.setCardHint(hint + "↑↓ スクロール   Esc 戻る")
}

type designShortcut struct {
	key   rune
	label string
	run   func()
}

func (u *boardUI) designShortcuts() []designShortcut {
	c := u.detailCard
	switch {
	case c.CanStartDesign():
		return []designShortcut{
			{'d', "設計から開始", func() { u.launchDetailPhase(board.DesignPhase, "") }},
		}
	case c.Status == board.AwaitingResultReview:
		return []designShortcut{{'a', "結果を承認して完了", u.approveDetailResult}}
	case c.CanReviewDesign():
		return []designShortcut{
			{'a', "承認して実装", func() { u.launchDetailPhase(board.ImplementationPhase, "") }},
			{'r', "コメントして再設計", u.editFeedback},
		}
	}
	return nil
}

func (u *boardUI) designKey(key rune) bool {
	for _, shortcut := range u.designShortcuts() {
		if key == shortcut.key {
			shortcut.run()
			return true
		}
	}
	return false
}

func (u *boardUI) approveDetailResult() {
	u.notice.SetText("")
	err := worker.ApproveResult(u.store, *u.detailCard)
	u.refresh()
	if err != nil {
		u.notice.SetText(err.Error())
	}
}

func (u *boardUI) editFeedback() {
	u.edit("")
	u.input.SetLabel("再設計のコメント > ")
	u.editingFeedback = true
}

func (u *boardUI) closeFeedbackEditor() {
	u.editingFeedback = false
	u.notice.SetText("")
	u.footer.SwitchToPage("keys")
	u.app.SetFocus(u.detail)
}

func (u *boardUI) launchDetailPhase(phase, feedback string) {
	u.notice.SetText("")
	err := worker.StartPhase(u.store, u.herdr, *u.detailCard, phase, feedback)
	u.refresh()
	if err != nil {
		u.notice.SetText(err.Error())
	}
}
