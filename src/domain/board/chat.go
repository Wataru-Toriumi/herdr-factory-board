package board

import (
	"fmt"
	"strings"
)

type ChatTurn struct {
	Question string
	Agent    string
	Reply    string `json:",omitempty"`
	Error    string `json:",omitempty"`
}

func (t ChatTurn) Pending() bool { return t.Reply == "" && t.Error == "" }

func (c Card) CanChat() bool { return c.manual() }

func (c Card) CanApplyChat() bool {
	return (c.CanReviewDesign() || c.CanReviewResult()) && len(c.Chat) > 0 && c.Chat[len(c.Chat)-1].Reply != ""
}

// 回答済みのやり取りだけを、次の相談や作業の文脈として渡す。
func (c Card) Discussion() string {
	var text strings.Builder
	for _, turn := range c.Chat {
		if turn.Reply != "" {
			fmt.Fprintf(&text, "あなた: %s\n%s: %s\n\n", turn.Question, AgentLabel(turn.Agent), turn.Reply)
		}
	}
	return strings.TrimSpace(text.String())
}
