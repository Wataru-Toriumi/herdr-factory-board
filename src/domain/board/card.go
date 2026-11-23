package board

import (
	"strconv"
	"strings"
)

const (
	Todo                 = "Todo"
	Starting             = "Starting"
	Running              = "In Progress"
	AwaitingDesignReview = "Awaiting Design Review"
	AwaitingResultReview = "Awaiting Result Review"
	Done                 = "Done"
	Failed               = "Failed"
	Canceled             = "Canceled"
)

type Card struct {
	ID                          int
	Prompt, Dir, Status, Source string
	Agent                       string `json:",omitempty"`
	Attempt                     int
	Pane, Result                string
	ScheduleID                  int    `json:",omitempty"`
	Phase                       string `json:",omitempty"`
	Design                      string `json:",omitempty"`
	Feedback                    string `json:",omitempty"`
	ImplementationFeedback      string `json:",omitempty"`
	PreviousResult              string `json:",omitempty"`
	ReopenCount                 int    `json:",omitempty"`
}

func (c Card) CanCancel() bool {
	switch c.Status {
	case Todo, Starting, Running, AwaitingDesignReview, AwaitingResultReview, Failed:
		return true
	}
	return false
}

func (c Card) SourceScheduleID() int {
	if c.ScheduleID != 0 {
		return c.ScheduleID
	}
	// 予定番号を独立して保存する前のカードでは、履歴の先頭に「定期 #番号」がある。
	source, ok := strings.CutPrefix(c.Source, "定期 #")
	if !ok {
		return 0
	}
	number, _, _ := strings.Cut(source, " ")
	id, err := strconv.Atoi(number)
	if err != nil || id < 1 {
		return 0
	}
	return id
}

func (c Card) CanConfigure() bool {
	return c.Status == Todo && c.manual()
}

func (c Card) manual() bool { return c.Source == "" && c.ScheduleID == 0 }

func (c Card) CanReopen() bool {
	return c.ReopenTarget() != ""
}

// ReopenTarget はカードを戻せるステータスを返す。対象外の場合は空文字を返す。
func (c Card) ReopenTarget() string {
	if !c.manual() {
		return ""
	}
	switch {
	case c.Status == Canceled:
		return Todo
	case c.Status == Done && c.Phase == ImplementationPhase:
		return AwaitingResultReview
	}
	return ""
}

func (c *Card) Reopen() bool {
	target := c.ReopenTarget()
	if target == "" {
		return false
	}
	if target == Todo {
		c.Phase, c.Pane = "", ""
	}
	c.Status = target
	// 同じ状態に戻った後も、別ペインに残っている古い操作を区別する。
	c.ReopenCount++
	return true
}
