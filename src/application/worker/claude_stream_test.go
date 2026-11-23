package worker

import (
	"strings"
	"testing"
)

func TestClaudeStreamKeepsProgressAcrossToolEvents(t *testing.T) {
	const events = `{"type":"assistant","message":{"content":[{"type":"text","text":"調査開始"},{"type":"text","text":"対象を確認"}]}}
{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Read"}]}}
`
	for _, tc := range []struct{ name, end, result, errorText string }{
		{"終了", `{"type":"result","subtype":"success","result":"調査完了"}`, "調査完了", ""},
		{"途中終了", "", "対象を確認", "Claude Code の終了結果を取得できませんでした"},
		{"不正なメッセージ", `{"type":"assistant","message":42}`, "対象を確認", "Claude Code のメッセージを読み取れません"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var pane strings.Builder
			got, err := readClaudeStream(strings.NewReader(events+tc.end), &pane)
			if got != tc.result {
				t.Fatalf("結果 = %q, 期待値 = %q", got, tc.result)
			}
			if tc.errorText == "" && err != nil {
				t.Fatal(err)
			}
			if tc.errorText != "" && (err == nil || !strings.Contains(err.Error(), tc.errorText)) {
				t.Fatalf("エラー = %v, 期待する文言 = %q", err, tc.errorText)
			}
			if got := pane.String(); got != "調査開始\n対象を確認\nツール: Read\n" {
				t.Fatalf("ペインの表示 = %q", got)
			}
		})
	}
}
