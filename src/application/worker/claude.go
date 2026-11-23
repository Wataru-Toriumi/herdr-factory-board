package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
)

func runClaude(ctx context.Context, task board.Card) (string, string) {
	args := []string{"-p", "--output-format", "stream-json", "--verbose"}
	if task.Phase == board.DesignPhase {
		args = append(args, "--tools", "Read,Glob,Grep", "--disallowedTools", "mcp__*")
	}
	cmd := agentCommand(ctx, task, "claude", args...)
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

// 担当ペインには経過を表示し、カードには最後の result イベントの本文を保存する。
func readClaudeStream(stdout io.Reader, progress io.Writer) (string, error) {
	dec := json.NewDecoder(stdout)
	var result, partial string
	var finished bool
	var resultErr error
	for {
		var event struct {
			Type, Subtype, Result string
			IsError               bool `json:"is_error"`
			Errors                []string
			Message               json.RawMessage
		}
		err := dec.Decode(&event)
		if err == io.EOF {
			if finished {
				return result, resultErr
			}
			return partial, errors.New("Claude Code の終了結果を取得できませんでした")
		}
		if err != nil {
			return partial, fmt.Errorf("Claude Code の出力を読み取れません: %w", err)
		}
		switch event.Type {
		case "assistant":
			var message struct {
				Content []struct{ Type, Text, Name string }
			}
			if err := json.Unmarshal(event.Message, &message); err != nil {
				return partial, fmt.Errorf("Claude Code のメッセージを読み取れません: %w", err)
			}
			for _, block := range message.Content {
				switch block.Type {
				case "text":
					partial = block.Text
					fmt.Fprintln(progress, block.Text)
				case "tool_use":
					fmt.Fprintln(progress, "ツール:", block.Name)
				}
			}
		case "result":
			finished, result = true, event.Result
			if event.IsError || event.Subtype != "success" {
				resultErr = fmt.Errorf("Claude Code の実行に失敗しました (%s)", event.Subtype)
				result = strings.TrimSpace(strings.Join(append([]string{result}, event.Errors...), "\n"))
			}
		}
	}
}
