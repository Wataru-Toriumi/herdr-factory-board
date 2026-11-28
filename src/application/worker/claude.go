package worker

import (
	"context"
	"errors"
	"io"
	"os"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
)

func runClaude(ctx context.Context, req agentRequest, lock *os.File) (string, string) {
	args := []string{"-p", "--output-format", "stream-json", "--verbose"}
	if req.readOnly {
		args = append(args, "--tools", "Read,Glob,Grep", "--disallowedTools", "mcp__*")
	}
	cmd := agentCommand(ctx, req, lock, "claude", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return board.Failed, err.Error()
	}
	if err := cmd.Start(); err != nil {
		return req.outcome(ctx, "", err)
	}
	result, streamErr := readClaudeStream(stdout, os.Stdout)
	// 不正なイベントがあってもパイプを最後まで読み、CLI が終了できるようにする。
	if streamErr != nil {
		_, _ = io.Copy(io.Discard, stdout)
	}
	return req.outcome(ctx, result, errors.Join(cmd.Wait(), streamErr))
}
