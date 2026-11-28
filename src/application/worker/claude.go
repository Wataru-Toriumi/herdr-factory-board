package worker

import (
	"context"
	"errors"
	"io"
	"os"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
)

func runClaude(ctx context.Context, task board.Card, lock *os.File) (string, string) {
	args := []string{"-p", "--output-format", "stream-json", "--verbose"}
	if task.Phase == board.DesignPhase || task.Phase == chatPhase {
		args = append(args, "--tools", "Read,Glob,Grep", "--disallowedTools", "mcp__*")
	}
	cmd := agentCommand(ctx, task, lock, "claude", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return board.Failed, err.Error()
	}
	if err := cmd.Start(); err != nil {
		return agentOutcome(ctx, task.Phase, "", err)
	}
	result, streamErr := readClaudeStream(stdout, os.Stdout)
	// 不正なイベントがあってもパイプを最後まで読み、CLI が終了できるようにする。
	if streamErr != nil {
		_, _ = io.Copy(io.Discard, stdout)
	}
	return agentOutcome(ctx, task.Phase, result, errors.Join(cmd.Wait(), streamErr))
}
