package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/application/worker"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/store"
)

// テストバイナリから起動した応答プロセスにも、本番と同じ入口を使う。
func TestMain(m *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == "chat-reply" {
		main()
		return
	}
	os.Exit(m.Run())
}

func TestTicketConversation(t *testing.T) {
	t.Setenv("HERDR_PLUGIN_STATE_DIR", t.TempDir())
	t.Setenv("HERDR_WORKSPACE_ID", "test")
	t.Setenv("HERDR_SOCKET_PATH", "socket")
	s, err := store.ForWorkspace()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))
	t.Setenv("CHAT_CAPTURE", filepath.Join(dir, "prompt"))
	t.Setenv("CHAT_WAIT", filepath.Join(dir, "continue"))
	for bin, script := range map[string]string{
		"codex": `test "$1" = exec
test "$4" = --sandbox
test "$5" = read-only
cat > "$CHAT_CAPTURE"
i=0
while [ ! -e "$CHAT_WAIT" ]; do
  i=$((i+1)); test "$i" -lt 400 || exit 1; sleep 0.02
done
printf '最初の回答' > "$3"`,
		"claude": `test "$*" = '-p --output-format stream-json --verbose --tools Read,Glob,Grep --disallowedTools mcp__*'
cat > "$CHAT_CAPTURE"
echo '{"type":"result","subtype":"success","result":"二回目の回答"}'`,
	} {
		if err := os.WriteFile(filepath.Join(dir, bin), []byte("#!/bin/sh\nset -eu\n"+script+"\n"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Edit(func(b *board.State) error {
		b.NewCard("元の依頼本文", dir, "")
		b.NewCard("他のチケットの依頼", dir, "")
		b.NewCard("定期実行", dir, "定期 #3")
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	read := func() board.State {
		t.Helper()
		b, err := s.Read()
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	waitReply := func(n int) board.Card {
		t.Helper()
		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			c := read().Cards[0]
			if len(c.Chat) == n && !c.Chat[n-1].Pending() {
				if c.Chat[n-1].Error != "" {
					t.Fatal(c.Chat[n-1].Error)
				}
				return c
			}
			time.Sleep(15 * time.Millisecond)
		}
		t.Fatal("担当から回答が届きません")
		return board.Card{}
	}
	original := read().Cards[0]
	if err := worker.SendChat(s, read().Cards[2], "定期"); err == nil {
		t.Fatal("定期実行に相談を送信できました")
	}
	if err := worker.SendChat(s, original, "初めの質問"); err != nil {
		t.Fatal(err)
	}
	if err := worker.SendChat(s, read().Cards[0], "同時送信"); err == nil {
		t.Fatal("回答中に別の質問を送信できました")
	}
	if err := os.WriteFile(os.Getenv("CHAT_WAIT"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	first := waitReply(1)
	if first.Chat[0].Reply != "最初の回答" {
		t.Fatal(first.Chat)
	}
	if err := s.ChangeCardAgent(first, board.Claude); err != nil {
		t.Fatal(err)
	}
	if err := worker.SendChat(s, read().Cards[0], "続きの質問"); err != nil {
		t.Fatal(err)
	}
	second := waitReply(2)
	if second.Chat[1].Reply != "二回目の回答" || second.Chat[1].Agent != board.Claude || len(read().Cards[1].Chat) != 0 {
		t.Fatal(second.Chat)
	}
	prompt, err := os.ReadFile(os.Getenv("CHAT_CAPTURE"))
	if err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{original.Prompt, "初めの質問", "最初の回答", "続きの質問"} {
		if !strings.Contains(string(prompt), part) {
			t.Fatalf("相談の入力に %s がありません", part)
		}
	}
	if strings.Contains(string(prompt), "他のチケットの依頼") {
		t.Fatal("他のチケットの文脈が混入しました")
	}
	second.Chat, second.Agent = nil, original.Agent
	if !second.Equal(original) {
		t.Fatalf("相談で元のチケットが変化しました: %+v", second)
	}
}
