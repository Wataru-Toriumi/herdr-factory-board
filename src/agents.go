package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

type livePane struct {
	ID        string `json:"pane_id"`
	Workspace string `json:"workspace_id"`
	Agent     string `json:"agent"`
	Status    string `json:"agent_status"`
	Title     string `json:"terminal_title_stripped"`
	Label     string `json:"label"`
}

type agentRow struct {
	pane, workspace, name, status, task string
	cardID                              int
}

func listPanes(herdr string) ([]livePane, error) {
	data, err := herdrList(herdr, "pane")
	if err != nil {
		return nil, err
	}
	var response struct {
		Result struct {
			Panes []livePane `json:"panes"`
		} `json:"result"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}
	if response.Result.Panes == nil {
		return nil, errors.New("Herdr の応答にペイン一覧がありません")
	}
	return response.Result.Panes, nil
}

func boardAgents(panes []livePane, spaces []workspace, cards []card) []agentRow {
	labels := make(map[string]string)
	for _, w := range spaces {
		labels[w.ID] = w.Label
	}
	workers := make(map[string]card)
	for _, c := range cards {
		if c.Status == running && c.Pane != "" {
			workers[c.Pane] = c
		}
	}
	var rows []agentRow
	for _, p := range panes {
		c, linked := workers[p.ID]
		if p.Agent == "" && !linked {
			continue
		}
		row := agentRow{pane: p.ID, workspace: labels[p.Workspace], name: p.Agent, status: p.Status, task: p.Title}
		if row.workspace == "" {
			row.workspace = p.Workspace
		}
		if row.task == "" {
			row.task = p.Label
		}
		if linked {
			row.cardID, row.task = c.ID, fmt.Sprintf("#%d %s", c.ID, c.Prompt)
			if row.name == "" {
				row.name = "Codex"
			}
			if row.status == "" || row.status == "unknown" {
				row.status = "working"
			}
		}
		rows = append(rows, row)
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].status == "blocked" && rows[j].status != "blocked" })
	return rows
}

func (row agentRow) focus(s store, herdr string) error {
	if row.cardID != 0 {
		return focusWorker(s, herdr, row.cardID)
	}
	out, err := exec.Command(herdr, "agent", "focus", row.pane).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
