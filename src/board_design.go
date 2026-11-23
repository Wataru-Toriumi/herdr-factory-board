package main

import "fmt"

func (u *boardUI) renderCardDetail(c card) {
	u.detailCard = &c
	body := fmt.Sprintf("#%d  %s\n作業先: %s\n%s", c.ID, c.Status, c.Dir, c.Prompt)
	if phase := c.phaseLabel(); phase != "" {
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
	hint := "↑↓ スクロール   Esc 戻る"
	switch {
	case c.canStartDesign():
		hint = "d 設計から開始   " + hint
	case c.Status == awaitingResultReview:
		hint = "a 結果を承認して完了   " + hint
	case c.canReviewDesign():
		hint = "a 承認して実装   r コメントして再設計   " + hint
	}
	if c.canChangeProject() {
		hint = "p 作業先を変更   " + hint
	}
	u.setCardHint(hint)
}

func (u *boardUI) designKey(key rune) bool {
	c := u.detailCard
	switch {
	case key == 'd' && c.canStartDesign():
		u.launchDetailPhase(designPhase, "")
	case key == 'a' && c.Status == awaitingResultReview:
		u.notice.SetText("")
		err := approveResult(u.store, *c)
		u.refresh()
		if err != nil {
			u.notice.SetText(err.Error())
		}
	case key == 'a' && c.canReviewDesign():
		u.launchDetailPhase(implementationPhase, "")
	case key == 'r' && c.canReviewDesign():
		u.edit("")
		u.input.SetLabel("再設計のコメント > ")
		u.editingFeedback = true
	default:
		return false
	}
	return true
}

func (u *boardUI) closeFeedbackEditor() {
	u.editingFeedback = false
	u.notice.SetText("")
	u.footer.SwitchToPage("keys")
	u.app.SetFocus(u.detail)
}

func (u *boardUI) launchDetailPhase(phase, feedback string) {
	u.notice.SetText("")
	err := launchFromDetail(u.store, u.herdr, *u.detailCard, phase, feedback)
	u.refresh()
	if err != nil {
		u.notice.SetText(err.Error())
	}
}
