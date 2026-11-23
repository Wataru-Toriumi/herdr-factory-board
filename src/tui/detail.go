package tui

import (
	"fmt"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
)

func (u *boardUI) openCardDetail(c board.Card) {
	u.openDetail("カードの詳細", cardDetailText(c), c.SourceScheduleID(), &c)
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
			u.detailBody = cardDetailText(card)
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
	hint := "↑↓ スクロール   Esc 戻る"
	if id := u.detailScheduleID; id != 0 {
		timing := "解除済み"
		if r, ok := u.schedules.byID(id); ok {
			timing = r.Frequency()
			hint = "t 定期実行の設定変更   " + hint
			if u.detailCard == nil {
				timing += "  担当: " + board.AgentLabel(r.Agent)
				hint = "g 担当を変更   " + hint
			}
		}
		text = fmt.Sprintf("定期実行 #%d  %s\n\n%s", id, timing, text)
	}
	if u.detail.GetText(false) != text {
		u.detail.SetText(text)
	}
	u.setCardHint(hint)
}

func cardDetailText(c board.Card) string {
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
	return body
}
