package tui

import (
	"github.com/Wataru-Toriumi/herdr-factory-board/src/application/worker"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
)

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
