package app

import (
	"strings"
	"testing"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/store"
	"github.com/gdamore/tcell/v2"
)

func TestChatInTicketDetail(t *testing.T) {
	for _, phase := range []string{board.DesignPhase, board.ImplementationPhase} {
		t.Run(phase, func(t *testing.T) {
			s := store.Store{Dir: t.TempDir()}
			err := s.Edit(func(b *board.State) error {
				c := b.NewCard("依頼本文", s.Dir, "")
				c.Phase, c.Design, c.Status = phase, "設計案", board.AwaitingDesignReview
				if phase == board.ImplementationPhase {
					c.Status, c.Result = board.AwaitingResultReview, "実装結果"
				}
				c.Chat = []board.ChatTurn{{Question: "なぜ？", Reply: "理由です"}}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			u := newBoardUI(s, "true")
			u.execute("result 1")
			key := func(r rune) { u.key(tcell.NewEventKey(tcell.KeyRune, r, 0)) }
			key('c')
			if u.activePage() != "chat" || !strings.Contains(u.detail.GetText(false), "理由です") {
				t.Fatal("保存された会話を開けません")
			}
			u.chatForm.Message.SetText("続き", true)
			u.refresh()
			if u.chatForm.Message.GetText() != "続き" || !strings.Contains(u.hint.GetText(false), "送信") {
				t.Fatal("更新時に入力中の会話が消えました")
			}
			u.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
			key('f')
			c := u.detail.Card()
			if c.Status != board.Starting || c.Phase != phase || c.AttemptChatTurns != 1 || c.Prompt != "依頼本文" {
				t.Fatalf("会話を反映した修正依頼: %+v", c)
			}
		})
	}
}

func TestChatRefreshPreservesDraftAndUpdatesReviewButtons(t *testing.T) {
	s := store.Store{Dir: t.TempDir()}
	if err := s.Edit(func(b *board.State) error {
		c := b.NewCard("依頼", s.Dir, "")
		c.Phase, c.Status, c.Design = board.DesignPhase, board.AwaitingDesignReview, "設計案"
		c.Chat = []board.ChatTurn{{Question: "相談"}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	u := newBoardUI(s, "true")
	u.execute("result 1")
	u.key(tcell.NewEventKey(tcell.KeyRune, 'c', 0))
	f := u.chatForm.Form
	applyLabel := "会話で設計を更新"
	if f.GetButtonIndex(applyLabel) >= 0 || f.GetButtonIndex("承認して実装") < 0 {
		t.Fatal("回答待ちの操作が正しく表示されていません")
	}
	u.chatForm.Message.SetText("下書き", true)
	f.SetFocus(f.GetFormItemCount() + f.GetButtonIndex("承認して実装"))
	u.app.SetFocus(f)
	if err := s.EditCard(1, func(c *board.Card) error {
		c.Chat[0].Reply = "回答"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	u.refresh()
	i := f.GetButtonIndex(applyLabel)
	if i < 0 || u.chatForm.Message.GetText() != "下書き" || u.app.GetFocus() != u.chatForm.Message {
		t.Fatal("回答後のボタン更新で入力欄またはフォーカスを復元できませんでした")
	}
	f.GetButton(i).InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), nil)
	if u.detail.Card().Status != board.Starting || f.GetButtonIndex(applyLabel) >= 0 {
		t.Fatal("更新したボタンから再設計を開始できませんでした")
	}
}
