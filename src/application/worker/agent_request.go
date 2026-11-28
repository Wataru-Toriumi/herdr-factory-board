package worker

import (
	"fmt"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
)

// agentRequest は各 CLI に渡す内容と、回答を受け取る際の条件を表す。
type agentRequest struct {
	agent, dir, prompt string
	readOnly           bool
	emptyResultError   string
}

func chatRequest(c board.Card, dir string) agentRequest {
	turn := c.Chat[len(c.Chat)-1]
	prompt := fmt.Sprintf("チケットについての相談です。作業先を読み取り専用で調査し、最後の質問に日本語で回答してください。実装やファイル変更は行わないでください。\n\n元の依頼\n%s\n\nカードの状態\n%s\n\n設計案\n%s\n\n実行結果\n%s\n\nこれまでの会話\n%s\n\n今回の質問\n%s",
		c.Prompt, c.Status, c.Design, c.Result, c.Discussion(), turn.Question)
	return agentRequest{
		agent: turn.Agent, dir: dir, prompt: prompt,
		readOnly: true, emptyResultError: "回答を取得できませんでした",
	}
}

func taskRequest(c board.Card) agentRequest {
	req := agentRequest{agent: c.Agent, dir: c.Dir, prompt: c.Prompt}
	switch c.Phase {
	case board.DesignPhase:
		req.readOnly = true
		req.emptyResultError = "設計案を取得できませんでした"
		req.prompt = "リポジトリを調査して設計案を作成してください。実装やファイル変更は行わず、変更方針、受け入れ条件、テスト方針、未決事項を最終メッセージにまとめてください。\n\n元の依頼\n" + c.Prompt
		if c.Design != "" {
			req.prompt += "\n\n前回の設計案\n" + c.Design + "\n\nレビューコメント\n" + c.Feedback
		}
	case board.ImplementationPhase:
		req.prompt = fmt.Sprintf("承認された設計案に沿って実装し、テスト結果を報告してください。\n\n元の依頼\n%s\n\n承認済みの設計案\n%s", c.Prompt, c.Design)
		if c.ImplementationFeedback != "" {
			req.prompt += "\n\n前回の実装結果\n" + c.PreviousResult + "\n\n以下のレビュー指摘に対応してください\n" + c.ImplementationFeedback
		}
	}
	// 作業を依頼した時点より後の相談は、進行中の作業に混ぜない。
	if c.AttemptChatTurns > 0 {
		c.Chat = c.Chat[:min(c.AttemptChatTurns, len(c.Chat))]
		if discussion := c.Discussion(); discussion != "" {
			req.prompt += "\n\n作業依頼時の会話（検討の経緯。設計案と最新の依頼を優先）\n" + discussion
		}
	}
	return req
}
