// Package agentlist はセッション内のエージェントと担当カードから一覧を組み立てる。
package agentlist

import (
	"fmt"
	"sort"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/herdr"
)

type Row struct {
	Pane                          string
	workspace, name, status, task string
	CardID                        int
}

type ViewModel struct {
	Rows    []Row
	Blocked int
}

func (m ViewModel) Title() string {
	if len(m.Rows) == 0 {
		return " エージェントなし "
	}
	return fmt.Sprintf(" エージェント %d ", len(m.Rows))
}

func Build(panes []herdr.Pane, spaces []herdr.Workspace, cards []board.Card) ViewModel {
	labels := make(map[string]string)
	for _, w := range spaces {
		labels[w.ID] = w.Label
	}
	workers := make(map[string]board.Card)
	for _, c := range cards {
		if pane := c.RunningPane(); pane != "" {
			workers[pane] = c
		}
	}
	var rows []Row
	blocked := 0
	for _, p := range panes {
		c, linked := workers[p.ID]
		if p.Agent == "" && !linked {
			continue
		}
		row := Row{Pane: p.ID, workspace: labels[p.Workspace], name: p.Agent, status: p.Status, task: p.Title}
		if row.workspace == "" {
			row.workspace = p.Workspace
		}
		if row.task == "" {
			row.task = p.Label
		}
		if linked {
			row.CardID, row.task = c.ID, fmt.Sprintf("#%d %s", c.ID, c.Prompt)
			if row.name == "" {
				row.name = board.AgentLabel(c.Agent)
			}
			if row.status == "" || row.status == "unknown" {
				row.status = "working"
			}
		}
		if row.status == "blocked" {
			blocked++
		}
		rows = append(rows, row)
	}
	sort.SliceStable(rows, func(i, j int) bool {
		return rows[i].status == "blocked" && rows[j].status != "blocked"
	})
	return ViewModel{Rows: rows, Blocked: blocked}
}

var agentStates = map[string]string{
	"blocked": "入力待ち", "working": "作業中", "idle": "待機中", "done": "完了",
}

func (r Row) Labels() (string, string) {
	state := agentStates[r.status]
	if state == "" {
		state = "状態不明"
	}
	return fmt.Sprintf("%s  ·  %s   %s", state, r.name, r.task), r.workspace + "   " + r.Pane
}
