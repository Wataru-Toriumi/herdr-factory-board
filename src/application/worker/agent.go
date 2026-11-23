package worker

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
)

func runAgent(ctx context.Context, task board.Card, resultPath string) (string, string) {
	switch task.Agent {
	case "", board.Codex:
		return runCodex(ctx, task, resultPath)
	case board.Claude:
		return runClaude(ctx, task)
	default:
		return board.Failed, fmt.Sprintf("未対応の担当エージェントです: %s", task.Agent)
	}
}

func agentCommand(ctx context.Context, task board.Card, bin string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, bin, args...)
	// エージェントが起動したコマンドも一緒に止められるよう、専用のプロセスグループで実行する。
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
	cmd.Dir = task.Dir
	cmd.Stdin = strings.NewReader(workerPrompt(task))
	cmd.Stderr = os.Stderr
	return cmd
}

func agentOutcome(ctx context.Context, phase, result string, runErr error) (string, string) {
	result = strings.TrimSpace(result)
	if ctx.Err() != nil {
		return board.Canceled, result
	}
	var exitErr *exec.ExitError
	if errors.As(runErr, &exitErr) && (exitErr.ExitCode() == 130 || exitErr.Sys().(syscall.WaitStatus).Signal() == syscall.SIGINT) {
		return board.Canceled, result
	}
	if runErr != nil {
		return board.Failed, strings.TrimSpace(result + "\n" + runErr.Error())
	}
	if result == "" {
		if phase == board.DesignPhase {
			return board.Failed, "設計案を取得できませんでした"
		}
		return board.Done, "終了コード 0（最終メッセージなし）"
	}
	return board.Done, result
}

func workerPrompt(c board.Card) string {
	switch c.Phase {
	case board.DesignPhase:
		prompt := "リポジトリを調査して設計案を作成してください。実装やファイル変更は行わず、変更方針、受け入れ条件、テスト方針、未決事項を最終メッセージにまとめてください。\n\n元の依頼\n" + c.Prompt
		if c.Design != "" {
			prompt += "\n\n前回の設計案\n" + c.Design + "\n\nレビューコメント\n" + c.Feedback
		}
		return prompt
	case board.ImplementationPhase:
		prompt := fmt.Sprintf("承認された設計案に沿って実装し、テスト結果を報告してください。\n\n元の依頼\n%s\n\n承認済みの設計案\n%s", c.Prompt, c.Design)
		if c.ImplementationFeedback != "" {
			prompt += "\n\n前回の実装結果\n" + c.PreviousResult + "\n\n以下のレビュー指摘に対応してください\n" + c.ImplementationFeedback
		}
		return prompt
	default:
		return c.Prompt
	}
}
