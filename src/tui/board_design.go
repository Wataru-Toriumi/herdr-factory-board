package tui

import (
	"github.com/Wataru-Toriumi/herdr-factory-board/src/application/worker"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
)

func (u *boardUI) cardShortcuts() []detailShortcut {
	c := u.detailCard
	if c == nil {
		return nil
	}
	var shortcuts []detailShortcut
	if c.CanCancel() {
		shortcuts = append(shortcuts, detailShortcut{'x', "キャンセル", u.cancelDetailCard})
	}
	if c.CanConfigure() {
		shortcuts = append(shortcuts, detailShortcut{'g', "担当を変更", func() { u.changeCardAgent(*c) }})
		shortcuts = append(shortcuts, detailShortcut{'p', "作業先を変更", func() { u.pickCardProject(*c) }})
	}
	switch {
	case c.CanStartDesign():
		return append(shortcuts, detailShortcut{'d', "設計から開始", func() { u.launchDetailPhase(board.DesignPhase, "") }})
	case c.CanReviewResult():
		return append(shortcuts,
			detailShortcut{'a', "結果を承認して完了", u.approveDetailResult},
			detailShortcut{'r', "修正を依頼", u.editFeedback},
		)
	case c.CanReviewDesign():
		return append(shortcuts,
			detailShortcut{'a', "承認して実装", func() { u.launchDetailPhase(board.ImplementationPhase, "") }},
			detailShortcut{'r', "コメントして再設計", u.editFeedback},
		)
	}
	switch c.ReopenTarget() {
	case board.Todo:
		return append(shortcuts, detailShortcut{'u', "未着手に戻す", u.reopenDetail})
	case board.AwaitingResultReview:
		return append(shortcuts, detailShortcut{'u', "承認を取り消す", u.reopenDetail})
	}
	return shortcuts
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
