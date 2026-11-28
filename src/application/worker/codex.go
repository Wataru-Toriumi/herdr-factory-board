package worker

import (
	"context"
	"os"
)

func runCodex(ctx context.Context, req agentRequest, lock *os.File, resultPath string) (string, string) {
	args := []string{"exec", "--output-last-message", resultPath}
	if req.readOnly {
		args = append(args, "--sandbox", "read-only")
	}
	cmd := agentCommand(ctx, req, lock, "codex", append(args, "-")...)
	cmd.Stdout = os.Stdout
	runErr := cmd.Run()
	msg, _ := os.ReadFile(resultPath)
	return req.outcome(ctx, string(msg), runErr)
}
