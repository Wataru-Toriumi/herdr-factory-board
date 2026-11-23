// Package herdr は Herdr CLI を通じてペインとワークスペースを操作する。
package herdr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Workspace struct {
	ID          string `json:"workspace_id"`
	Label       string `json:"label"`
	Focused     bool   `json:"focused"`
	AgentStatus string `json:"agent_status"`
}

func list(bin, resource string, response any) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, bin, resource, "list").Output()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(out, response)
}

func ListWorkspaces(bin string) ([]Workspace, error) {
	var response struct {
		Result struct {
			Workspaces []Workspace `json:"workspaces"`
		} `json:"result"`
	}
	if err := list(bin, "workspace", &response); err != nil {
		return nil, err
	}
	workspaces := response.Result.Workspaces
	if workspaces == nil {
		return nil, fmt.Errorf("Herdr の応答にワークスペース一覧がありません")
	}
	return workspaces, nil
}

func OpenPlugin(bin, entry string, env ...string) error {
	args := []string{"plugin", "pane", "open", "--plugin", os.Getenv("HERDR_PLUGIN_ID"), "--entrypoint", entry, "--placement", "tab", "--workspace", os.Getenv("HERDR_WORKSPACE_ID"), "--no-focus"}
	for _, v := range env {
		args = append(args, "--env", v)
	}
	out, err := exec.Command(bin, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

type Pane struct {
	ID        string `json:"pane_id"`
	Workspace string `json:"workspace_id"`
	Agent     string `json:"agent"`
	Status    string `json:"agent_status"`
	Title     string `json:"terminal_title_stripped"`
	Label     string `json:"label"`
}

func ListPanes(bin string) ([]Pane, error) {
	var response struct {
		Result struct {
			Panes []Pane `json:"panes"`
		} `json:"result"`
	}
	if err := list(bin, "pane", &response); err != nil {
		return nil, err
	}
	if response.Result.Panes == nil {
		return nil, errors.New("Herdr の応答にペイン一覧がありません")
	}
	return response.Result.Panes, nil
}

func FocusPane(bin, pane string) error {
	return exec.Command(bin, "plugin", "pane", "focus", pane).Run()
}

func FocusAgent(bin, pane string) error {
	out, err := exec.Command(bin, "agent", "focus", pane).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
