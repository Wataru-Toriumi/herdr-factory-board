package app

import (
	"strings"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/application/scheduler"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/application/worker"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/schedule"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/tui/detail"
)

func (u *boardUI) openCardDetail(c board.Card) {
	u.openDetail("カードの詳細", detail.CardText(c), c.SourceScheduleID(), &c)
}

func (u *boardUI) openDetail(title, body string, scheduleID int, card *board.Card) {
	u.detail.Open(title, body, scheduleID, card)
	u.pages.SwitchToPage("detail")
	u.app.SetFocus(u.detail)
}

func (u *boardUI) runDetailAction(action detail.Action) {
	switch action {
	case detail.OpenChat:
		u.openChat()
	case detail.ApplyChat:
		u.launchDetailPhase(u.detail.Card().Phase, "チケットの会話で合意した内容を反映してください。")
	case detail.CancelCard:
		u.cancelDetailCard()
	case detail.ChangeCardAgent:
		u.changeCardAgent(*u.detail.Card())
	case detail.ChangeProject:
		u.pickCardProject(*u.detail.Card())
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
		u.withDetailSchedule(func(r schedule.Rule) {
			u.finishDetailAction(scheduler.ChangeAgent(u.store, r.ID, r.Agent, board.NextAgent(r.Agent)))
		})
	case detail.EditTiming:
		u.withDetailSchedule(u.editScheduleTiming)
	case detail.EditScheduleSettings:
		u.withDetailSchedule(func(r schedule.Rule) { u.openScheduleForm(r, r.TimingKind()) })
	}
}

func (u *boardUI) withDetailSchedule(action func(schedule.Rule)) {
	if r, ok := u.schedules.ByID(u.detail.ScheduleID()); ok {
		action(r)
	}
}

func (u *boardUI) cancelDetailCard() {
	err := worker.Cancel(u.store, *u.detail.Card())
	u.refresh()
	if err != nil {
		u.notice.SetText(err.Error())
	}
}

func (u *boardUI) approveDetailResult() {
	u.finishDetailAction(worker.ApproveResult(u.store, *u.detail.Card()))
}

func (u *boardUI) reopenDetail() {
	u.finishDetailAction(worker.Reopen(u.store, *u.detail.Card()))
}

func (u *boardUI) launchDetailPhase(phase, feedback string) {
	u.finishDetailAction(worker.StartPhase(u.store, u.herdr, *u.detail.Card(), phase, feedback))
}

func (u *boardUI) finishDetailAction(err error) {
	u.notice.SetText("")
	u.refresh()
	if err != nil {
		u.notice.SetText(err.Error())
	}
}

func (u *boardUI) changeCardAgent(seen board.Card) {
	u.finishDetailAction(u.store.ChangeCardAgent(seen, board.NextAgent(seen.Agent)))
}

func (u *boardUI) editFeedback() {
	label := "再設計のコメント > "
	if u.detail.Card().CanReviewResult() {
		label = "実装の修正内容 > "
	}
	u.openInput(label, "", func(value string) {
		if strings.TrimSpace(value) == "" {
			u.notice.SetText("修正内容を入力してください")
			return
		}
		u.closeFeedbackEditor()
		u.launchDetailPhase(u.detail.Card().Phase, value)
	}, func() {
		u.closeFeedbackEditor()
		u.refresh()
	})
}

func (u *boardUI) closeFeedbackEditor() {
	u.editor = inputSession{}
	u.notice.SetText("")
	u.footer.SwitchToPage("keys")
	u.app.SetFocus(u.detail)
}
