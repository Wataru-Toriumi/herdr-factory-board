package tui

import (
	"github.com/Wataru-Toriumi/herdr-factory-board/src/application/scheduler"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/tui/detail"
)

func (u *boardUI) detailKey(key rune) bool {
	// 予定が解除されても、定期実行の操作キーは本文のスクロールに渡さない。
	for _, shortcut := range detail.Shortcuts(u.detailCard, true) {
		if shortcut.Key == key {
			u.runDetailAction(shortcut.Action)
			return true
		}
	}
	return false
}

func (u *boardUI) runDetailAction(action detail.Action) {
	switch action {
	case detail.CancelCard:
		u.cancelDetailCard()
	case detail.ChangeCardAgent:
		u.changeCardAgent(*u.detailCard)
	case detail.ChangeProject:
		u.pickCardProject(*u.detailCard)
	case detail.StartDesign:
		u.launchDetailPhase(board.DesignPhase, "")
	case detail.ApproveResult:
		u.approveDetailResult()
	case detail.EditFeedback:
		u.editFeedback()
	case detail.StartImplementation:
		u.launchDetailPhase(board.ImplementationPhase, "")
	case detail.ReopenCard:
		u.reopenDetail()
	case detail.ChangeScheduleAgent:
		if r, ok := u.schedules.byID(u.detailScheduleID); ok {
			u.finishDetailAction(scheduler.ChangeAgent(u.store, r.ID, r.Agent, nextAgent(r.Agent)))
		}
	case detail.EditTiming:
		if r, ok := u.schedules.byID(u.detailScheduleID); ok {
			u.editScheduleTiming(r)
		}
	}
}
