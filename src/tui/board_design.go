package tui

import (
	"fmt"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/application/worker"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
)

func (u *boardUI) renderCardDetail(c board.Card) {
	u.detailCard = &c
	body := fmt.Sprintf("#%d  %s\n担当: %s\n作業先: %s\n%s", c.ID, c.Status, board.AgentLabel(c.Agent), c.Dir, c.Prompt)
	if phase := c.PhaseLabel(); phase != "" {
		body += "\n\n" + phase
	}
	if c.Design != "" {
		body += "\n\n設計案\n" + c.Design
	}
	if c.Feedback != "" {
		body += "\n\nレビューコメント\n" + c.Feedback
	}
	if c.ImplementationFeedback != "" {
		if c.PreviousResult != "" {
			body += "\n\n前回の実装結果\n" + c.PreviousResult
		}
		body += "\n\n実装の修正依頼\n" + c.ImplementationFeedback
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
	u.setCardHint("↑↓ スクロール   Esc 戻る")
}

type cardShortcut struct {
	key   rune
	label string
	run   func()
}

func (u *boardUI) cardShortcuts() []cardShortcut {
	c := u.detailCard
	if c == nil {
		return nil
	}
	var shortcuts []cardShortcut
	if c.CanCancel() {
		shortcuts = append(shortcuts, cardShortcut{'x', "キャンセル", u.cancelDetailCard})
	}
	if c.CanConfigure() {
		shortcuts = append(shortcuts, cardShortcut{'g', "担当を変更", func() { u.changeCardAgent(*c) }})
		shortcuts = append(shortcuts, cardShortcut{'p', "作業先を変更", func() { u.pickCardProject(*c) }})
	}
	switch {
	case c.CanStartDesign():
		return append(shortcuts, cardShortcut{'d', "設計から開始", func() { u.launchDetailPhase(board.DesignPhase, "") }})
	case c.CanReviewResult():
		return append(shortcuts,
			cardShortcut{'a', "結果を承認して完了", u.approveDetailResult},
			cardShortcut{'r', "修正を依頼", u.editFeedback},
		)
	case c.CanReviewDesign():
		return append(shortcuts,
			cardShortcut{'a', "承認して実装", func() { u.launchDetailPhase(board.ImplementationPhase, "") }},
			cardShortcut{'r', "コメントして再設計", u.editFeedback},
		)
	}
	switch c.ReopenTarget() {
	case board.Todo:
		return append(shortcuts, cardShortcut{'u', "未着手に戻す", u.reopenDetail})
	case board.AwaitingResultReview:
		return append(shortcuts, cardShortcut{'u', "承認を取り消す", u.reopenDetail})
	}
	return shortcuts
}

func (u *boardUI) setCardHint(suffix string) {
	hint := ""
	for _, shortcut := range u.cardShortcuts() {
		hint += string(shortcut.key) + " " + shortcut.label + "   "
	}
	u.hint.SetText(hint + suffix)
}

func (u *boardUI) cardKey(key rune) bool {
	for _, shortcut := range u.cardShortcuts() {
		if key == shortcut.key {
			shortcut.run()
			return true
		}
	}
	return false
}

func (u *boardUI) cancelDetailCard() {
	err := worker.Cancel(u.store, *u.detailCard)
	u.refresh()
	if err != nil {
		u.notice.SetText(err.Error())
	}
}

func (u *boardUI) approveDetailResult() {
	u.finishDetailAction(worker.ApproveResult(u.store, *u.detailCard))
}

func (u *boardUI) reopenDetail() {
	u.finishDetailAction(worker.Reopen(u.store, *u.detailCard))
}

func (u *boardUI) editFeedback() {
	u.edit("")
	if u.detailCard.CanReviewResult() {
		u.input.SetLabel("実装の修正内容 > ")
	} else {
		u.input.SetLabel("再設計のコメント > ")
	}
	u.editingFeedback = true
}

func (u *boardUI) closeFeedbackEditor() {
	u.editingFeedback = false
	u.notice.SetText("")
	u.footer.SwitchToPage("keys")
	u.app.SetFocus(u.detail)
}

func (u *boardUI) launchDetailPhase(phase, feedback string) {
	u.finishDetailAction(worker.StartPhase(u.store, u.herdr, *u.detailCard, phase, feedback))
}

func (u *boardUI) finishDetailAction(err error) {
	u.notice.SetText("")
	u.refresh()
	if err != nil {
		u.notice.SetText(err.Error())
	}
}

func nextAgent(current string) string {
	if current == board.Claude {
		return board.Codex
	}
	return board.Claude
}

func (u *boardUI) changeCardAgent(seen board.Card) {
	u.finishDetailAction(u.store.ChangeCardAgent(seen, nextAgent(seen.Agent)))
}
