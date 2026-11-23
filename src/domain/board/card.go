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

func (c Card) CanChangeProject() bool {
	return c.Status == Todo && c.manual()
}

func (c Card) manual() bool { return c.Source == "" && c.ScheduleID == 0 }

func (c Card) CanReviewResult() bool {
	return c.manual() && c.Phase == ImplementationPhase && c.Status == AwaitingResultReview
}

func (c Card) CanReopen() bool {
	return c.manual() && (c.Status == Canceled || (c.Status == Done && c.Phase == ImplementationPhase))
}

const (
	DesignPhase         = "design"
	ImplementationPhase = "implementation"
)

func (c Card) CanStartDesign() bool {
	return c.manual() && (c.Status == Todo || (c.Status == Failed && c.Phase == DesignPhase))
}

func (c Card) CanReviewDesign() bool {
	return c.manual() && c.Status == AwaitingDesignReview && c.Phase == DesignPhase && c.Design != ""
}

func (c Card) PhaseLabel() string {
	switch c.Status {
	case Canceled:
		return "キャンセル済み"
	case Failed:
		switch c.Phase {
		case DesignPhase:
			return "設計失敗"
		case ImplementationPhase:
			return "実装失敗"
		}
		return "失敗"
	case AwaitingDesignReview:
		return "設計完了（承認待ち）"
	case AwaitingResultReview:
		return "実装結果の確認待ち"
	}
	switch c.Phase {
	case DesignPhase:
		return "設計中"
	case ImplementationPhase:
		if c.Status == Done {
			return "実装完了"
		}
		return "実装中"
	}
	return ""
}
