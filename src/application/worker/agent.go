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

func runAgent(ctx context.Context, req agentRequest, lock *os.File, resultPath string) (string, string) {
	switch req.agent {
	case "", board.Codex:
		return runCodex(ctx, req, lock, resultPath)
	case board.Claude:
		return runClaude(ctx, req, lock)
	default:
		return board.Failed, fmt.Sprintf("未対応の担当エージェントです: %s", req.agent)
	}
}

func agentCommand(ctx context.Context, req agentRequest, lock *os.File, bin string, args ...string) *exec.Cmd {
	// 待機するシェルにも同じロックを保持させ、ワーカーが強制終了しても
	// エージェントの終了までは復旧を許可しない。CLI やその子にはロックを渡さない。
	path, lookupErr := exec.LookPath(bin)
	cmd := exec.CommandContext(ctx, "/bin/sh", append([]string{"-c", `"$@" 3>&-; exit $?`, "factory-board", path}, args...)...)
	if lookupErr != nil {
		cmd.Err = lookupErr
	}
	cmd.ExtraFiles = []*os.File{lock}
	// エージェントが起動したコマンドも一緒に止められるよう、専用のプロセスグループで実行する。
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
	cmd.Dir = req.dir
	cmd.Stdin = strings.NewReader(req.prompt)
	cmd.Stderr = os.Stderr
	return cmd
}

func (req agentRequest) outcome(ctx context.Context, result string, runErr error) (string, string) {
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
		if req.emptyResultError != "" {
			return board.Failed, req.emptyResultError
		}
		return board.Done, "終了コード 0（最終メッセージなし）"
	}
	return board.Done, result
}
