package worker

import (
	"context"
	"os"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
)

func runCodex(ctx context.Context, task board.Card, resultPath string) (string, string) {
	args := []string{"exec", "--output-last-message", resultPath}
	if task.Phase == board.DesignPhase {
		args = append(args, "--sandbox", "read-only")
	}
	cmd := agentCommand(ctx, task, "codex", append(args, "-")...)
	cmd.Stdout = os.Stdout
	runErr := cmd.Run()
	msg, _ := os.ReadFile(resultPath)
	return agentOutcome(ctx, task.Phase, string(msg), runErr)
}
