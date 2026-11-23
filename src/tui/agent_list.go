package tui

import "fmt"

type agentList struct {
	rowList[agentRow]
}

func (l *agentList) blockedCount() int {
	blocked := 0
	for _, r := range l.rows {
		if r.status == "blocked" {
			blocked++
		}
	}
	return blocked
}

var agentStates = map[string]string{
	"blocked": "入力待ち", "working": "作業中", "idle": "待機中", "done": "完了",
}

func agentRowLabels(r agentRow) (string, string) {
	state := agentStates[r.status]
	if state == "" {
		state = "状態不明"
	}
	return fmt.Sprintf("%s  ·  %s   %s", state, r.name, r.task), r.workspace + "   " + r.pane
}

func (l *agentList) render(rows []agentRow) {
	l.setRows(rows, func(a, b agentRow) bool { return a.pane == b.pane }, agentRowLabels)
	l.SetTitle(fmt.Sprintf(" エージェント %d ", len(rows)))
	if len(rows) == 0 {
		l.SetTitle(" エージェントなし ")
	}
}
