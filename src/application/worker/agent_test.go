package worker

import (
	"context"
	"errors"
	"strings"

	"testing"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
)

func TestCanceledAgentOutcomeKeepsPartialMessage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	status, result := agentOutcome(ctx, board.DesignPhase, "途中までの設計案\n", errors.New("signal: killed"))
	if status != board.Canceled || result != "途中までの設計案" {
		t.Fatalf("中止後の結果: %q, %q", status, result)
	}
}

func TestPhasePromptIncludesOnlyReviewedChat(t *testing.T) {
	for _, phase := range []string{"", board.DesignPhase, board.ImplementationPhase} {
		c := board.Card{Phase: phase, Prompt: "依頼本文", Design: "設計案", AttemptChatTurns: 1,
			Chat: []board.ChatTurn{{Question: "最初の質問", Reply: "合意した内容"}, {Question: "実行開始後の質問", Reply: "後からの回答"}}}
		prompt := workerPrompt(c)
		if !strings.Contains(prompt, "合意した内容") || strings.Contains(prompt, "後からの回答") {
			t.Fatalf("%s の作業に渡した会話が異なります: %s", phase, prompt)
		}
	}
}
