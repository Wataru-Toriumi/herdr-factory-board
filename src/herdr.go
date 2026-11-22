package main

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

type workspace struct {
	ID          string `json:"workspace_id"`
	Label       string `json:"label"`
	Focused     bool   `json:"focused"`
	AgentStatus string `json:"agent_status"`
}

func herdrList(herdr, resource string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, herdr, resource, "list").Output()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return out, err
}

func listWorkspaces(herdr string) ([]workspace, error) {
	output, err := herdrList(herdr, "workspace")
	if err != nil {
		return nil, err
	}

	var response struct {
		Result struct {
			Workspaces []workspace `json:"workspaces"`
		} `json:"result"`
	}
	if err := json.Unmarshal(output, &response); err != nil {
		return nil, err
	}
	workspaces := response.Result.Workspaces
	if workspaces == nil {
		return nil, fmt.Errorf("Herdr の応答にワークスペース一覧がありません")
	}
	return workspaces, nil
}

func openPlugin(herdr, entry string, env ...string) error {
	args := []string{"plugin", "pane", "open", "--plugin", os.Getenv("HERDR_PLUGIN_ID"), "--entrypoint", entry, "--placement", "tab", "--workspace", os.Getenv("HERDR_WORKSPACE_ID"), "--no-focus"}
	for _, v := range env {
		args = append(args, "--env", v)
	}
	out, err := exec.Command(herdr, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func focusWorker(s store, herdr string, id int) error {
	b, err := s.read()
	if err != nil {
		return err
	}
	c, err := b.find(id)
	if err != nil {
		return err
	}
	if c.Status != running || c.Pane == "" {
		return errors.New("実行中の担当ペインがありません。結果は result で確認できます")
	}
	return exec.Command(herdr, "plugin", "pane", "focus", c.Pane).Run()
}
