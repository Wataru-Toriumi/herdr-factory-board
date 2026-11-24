package detail

import (
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/schedule"
	"github.com/rivo/tview"
)

type View struct {
	*tview.TextView
	card       *board.Card
	scheduleID int
	body       string
	lookup     func(int) (schedule.Rule, bool)
	showHint   func(string)
}

func NewView(lookup func(int) (schedule.Rule, bool), showHint func(string)) *View {
	v := &View{TextView: tview.NewTextView().SetScrollable(true).SetWordWrap(true), lookup: lookup, showHint: showHint}
	v.SetBorder(true).SetTitleAlign(tview.AlignLeft)
	return v
}

func (v *View) Card() *board.Card { return v.card }
func (v *View) ScheduleID() int   { return v.scheduleID }

func (v *View) ClearSelection() {
	v.card = nil
	v.scheduleID = 0
	v.body = ""
}

func (v *View) Open(title, body string, scheduleID int, card *board.Card) {
	v.body, v.scheduleID, v.card = body, scheduleID, card
	v.SetTitle(" " + title + " ")
	v.Render()
	v.ScrollToBeginning()
}

func (v *View) Refresh(b board.State) {
	if v.card != nil {
		if c, err := b.FindCard(v.card.ID); err == nil && *c != *v.card {
			card := *c
			v.card = &card
			v.body = CardText(card)
			v.Render()
			return
		}
	}
	// カードが同じでも、元の予定が別ペインで変更・解除されることがある。
	if v.scheduleID != 0 {
		v.Render()
	}
}

func (v *View) Render() {
	text := v.body
	var rule *schedule.Rule
	if id := v.scheduleID; id != 0 {
		if r, ok := v.lookup(id); ok {
			rule = &r
			if v.card == nil {
				v.body = RuleBody(r)
			}
		}
		text = ScheduleText(v.body, id, rule, v.card == nil)
	}
	if v.GetText(false) != text {
		v.SetText(text)
	}
	v.showHint(Shortcuts(v.card, rule != nil).Hint())
}

func (v *View) HandleKey(key rune, run func(Action)) bool {
	// 予定が解除されても、定期実行の操作キーは本文のスクロールに渡さない。
	for _, shortcut := range Shortcuts(v.card, true) {
		if shortcut.Key == key {
			run(shortcut.Action)
			return true
		}
	}
	return false
}
