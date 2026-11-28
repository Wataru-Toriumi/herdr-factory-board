// Package detail は詳細画面の本文と、状態に応じた操作案内を組み立てる。
package detail

import (
	"fmt"
	"strings"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/schedule"
)

func RuleBody(r schedule.Rule) string {
	return fmt.Sprintf("作業先: %s\n\n%s", r.Dir, r.Prompt)
}

func ScheduleText(body string, id int, rule *schedule.Rule, showAgent bool) string {
	timing := "解除済み"
	if rule != nil {
		timing = rule.Frequency()
		if showAgent {
			timing += "  担当: " + board.AgentLabel(rule.Agent)
		}
	}
	return fmt.Sprintf("定期実行 #%d  %s\n\n%s", id, timing, body)
}

func CardText(c board.Card) string {
	body := fmt.Sprintf("#%d  %s\n担当: %s\n作業先: %s\n", c.ID, c.Status, board.AgentLabel(c.Agent), c.Dir)
	if c.CanChat() {
		body += "\n元の依頼\n"
	}
	body += c.Prompt
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
	if c.CanChat() {
		var thread strings.Builder
		thread.WriteString("\n\n会話")
		if len(c.Chat) == 0 {
			thread.WriteString("\nまだ会話はありません。c で担当に相談できます。")
		}
		for _, turn := range c.Chat {
			answer := turn.Reply
			switch {
			case turn.Pending():
				answer = "回答待ち…"
			case turn.Error != "":
				answer = "回答エラー: " + turn.Error
			}
			fmt.Fprintf(&thread, "\n\nあなた\n%s\n\n%s\n%s", turn.Question, board.AgentLabel(turn.Agent), answer)
		}
		body += thread.String()
	}
	return body
}
