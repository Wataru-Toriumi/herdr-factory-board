package tui

import (
	"fmt"
	"sort"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/application/worker"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/herdr"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/store"
)

type agentRow struct {
	pane, workspace, name, status, task string
	cardID                              int
}

func boardAgents(panes []herdr.Pane, spaces []herdr.Workspace, cards []board.Card) []agentRow {
	labels := make(map[string]string)
	for _, w := range spaces {
		labels[w.ID] = w.Label
	}
	workers := make(map[string]board.Card)
	for _, c := range cards {
		if c.Status == board.Running && c.Pane != "" {
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

func (row agentRow) focus(s store.Store, bin string) error {
	if row.cardID != 0 {
		return worker.Focus(s, bin, row.cardID)
	}
	return herdr.FocusAgent(bin, row.pane)
}
