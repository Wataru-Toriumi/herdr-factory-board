package worker

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
)

func TestClaudeCommand(t *testing.T) {
	bin := t.TempDir()
	script := `#!/bin/sh
set -eu
test "$1" = -p && test "$2" = --output-format && test "$3" = stream-json && test "$4" = --verbose || exit 2
shift 4
if test "$PHASE" = design; then
	test "$*" = '--tools Read,Glob,Grep --disallowedTools mcp__*' || exit 3
else
	test "$#" = 0 || exit 4
fi
cat > prompt
echo '{"type":"system","subtype":"init"}'
echo '{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Read"},{"type":"text","text":"調査中"}]}}'
case "$MODE" in
	interrupt) exit 130 ;;
	error) echo '{"type":"result","subtype":"error_during_execution","is_error":true,"errors":["認証エラー"]}' ;;
	truncated) exit 0 ;;
	*) echo '{"type":"result","subtype":"success","is_error":false,"result":"完了報告"}' ;;
esac
`
	if err := os.WriteFile(filepath.Join(bin, "claude"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+":"+os.Getenv("PATH"))
	for _, tc := range []struct{ phase, mode, status, result, prompt string }{
		{"", "success", board.Done, "完了報告", "元のタスク"},
		{board.DesignPhase, "success", board.Done, "完了報告", "設計案を作成"},
		{board.ImplementationPhase, "success", board.Done, "完了報告", "承認済みの案"},
		{"", "error", board.Failed, "認証エラー", "元のタスク"},
		{"", "interrupt", board.Canceled, "調査中", "元のタスク"},
		{"", "truncated", board.Failed, "終了結果を取得できません", "元のタスク"},
	} {
		t.Run(tc.phase+"/"+tc.mode, func(t *testing.T) {
			task := board.Card{Agent: board.Claude, Dir: t.TempDir(), Phase: tc.phase, Prompt: "元のタスク", Design: "承認済みの案"}
			t.Setenv("PHASE", tc.phase)
			t.Setenv("MODE", tc.mode)
			lock, err := os.CreateTemp(t.TempDir(), "lock")
			if err != nil {
				t.Fatal(err)
			}
			defer lock.Close()
			status, result := runAgent(context.Background(), taskRequest(task), lock, "")
			if status != tc.status || !strings.Contains(result, tc.result) {
				t.Fatalf("状態 = %q, 結果 = %q", status, result)
			}
			prompt, err := os.ReadFile(filepath.Join(task.Dir, "prompt"))
			if err != nil || !strings.Contains(string(prompt), tc.prompt) {
				t.Fatalf("プロンプト = %q, %v", prompt, err)
			}
		})
	}
}
