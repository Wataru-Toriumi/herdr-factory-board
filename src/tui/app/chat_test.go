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
			u.chatInput.SetText("続き", true)
			u.refresh()
			if u.chatInput.GetText() != "続き" || !strings.Contains(u.hint.GetText(false), "送信") {
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
