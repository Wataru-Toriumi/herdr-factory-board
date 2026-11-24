package tui

import (
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/schedule"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/tui/detail"
)

func (u *boardUI) openCardDetail(c board.Card) {
	u.openDetail("カードの詳細", detail.CardText(c), c.SourceScheduleID(), &c)
}

func (u *boardUI) openDetail(title, body string, scheduleID int, card *board.Card) {
	u.detailBody, u.detailScheduleID, u.detailCard = body, scheduleID, card
	u.detail.SetTitle(" " + title + " ")
	u.renderDetail()
	u.detail.ScrollToBeginning()
	u.pages.SwitchToPage("detail")
	u.app.SetFocus(u.detail)
}

func (u *boardUI) refreshDetail(b board.State) {
	if u.detailCard != nil {
		if c, err := b.FindCard(u.detailCard.ID); err == nil && *c != *u.detailCard {
			card := *c
			u.detailCard = &card
			u.detailBody = detail.CardText(card)
			u.renderDetail()
			return
		}
	}
	// カードが同じでも、元の予定が別ペインで変更・解除されることがある。
	if u.detailScheduleID != 0 {
		u.renderDetail()
	}
}

func (u *boardUI) renderDetail() {
	text := u.detailBody
	var rule *schedule.Rule
	if id := u.detailScheduleID; id != 0 {
		if r, ok := u.schedules.model.ByID(id); ok {
			rule = &r
			if u.detailCard == nil {
				u.detailBody = detail.RuleBody(r)
			}
		}
		text = detail.ScheduleText(u.detailBody, id, rule, u.detailCard == nil)
	}
	if u.detail.GetText(false) != text {
		u.detail.SetText(text)
	}
	u.hint.SetText(detail.Shortcuts(u.detailCard, rule != nil).Hint())
}
