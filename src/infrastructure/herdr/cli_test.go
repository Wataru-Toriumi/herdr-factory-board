package herdr

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestListResponses(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "herdr")
	script := "#!/bin/sh\nprintf '%s' \"$LIST_OUTPUT\"\nexit \"$LIST_EXIT\"\n"
	if err := os.WriteFile(bin, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LIST_EXIT", "0")
	t.Setenv("LIST_OUTPUT", `{"result":{"panes":[],"workspaces":[]}}`)
	panes, err := ListPanes(bin)
	if err != nil || panes == nil || len(panes) != 0 {
		t.Fatalf("空のペイン一覧: %v, %v", panes, err)
	}
	spaces, err := ListWorkspaces(bin)
	if err != nil || spaces == nil || len(spaces) != 0 {
		t.Fatalf("空のワークスペース一覧: %v, %v", spaces, err)
	}

	t.Setenv("LIST_OUTPUT", `{"result":{}}`)
	if _, err := ListPanes(bin); err == nil || err.Error() != "Herdr の応答にペイン一覧がありません" {
		t.Fatalf("ペイン一覧の欠落: %v", err)
	}
	if _, err := ListWorkspaces(bin); err == nil || err.Error() != "Herdr の応答にワークスペース一覧がありません" {
		t.Fatalf("ワークスペース一覧の欠落: %v", err)
	}

	t.Setenv("LIST_OUTPUT", "{")
	var syntaxErr *json.SyntaxError
	if _, err := ListPanes(bin); !errors.As(err, &syntaxErr) {
		t.Fatalf("不正な JSON: %v", err)
	}
	t.Setenv("LIST_EXIT", "7")
	var exitErr *exec.ExitError
	if _, err := ListPanes(bin); !errors.As(err, &exitErr) || exitErr.ExitCode() != 7 {
		t.Fatalf("CLI の異常終了: %v", err)
	}
}
