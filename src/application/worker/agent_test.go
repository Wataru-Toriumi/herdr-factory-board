package worker

import (
	"context"
	"errors"

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
