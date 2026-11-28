package worker

import (
	"context"
	"os"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
)

func runCodex(ctx context.Context, task board.Card, lock *os.File, resultPath string) (string, string) {
	args := []string{"exec", "--output-last-message", resultPath}
	if task.Phase == board.DesignPhase || task.Phase == chatPhase {
		args = append(args, "--sandbox", "read-only")
	}
	cmd := agentCommand(ctx, task, lock, "codex", append(args, "-")...)
	cmd.Stdout = os.Stdout
	runErr := cmd.Run()
	msg, _ := os.ReadFile(resultPath)
	return agentOutcome(ctx, task.Phase, string(msg), runErr)
}
