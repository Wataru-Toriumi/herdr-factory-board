package taskboard

import "github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"

const (
	Todo = iota
	Design
	DesignComplete
	Work
	Review
	Done
	FailedOrCanceled
	LaneCount
)

func LaneFor(c board.Card) int {
	switch c.Status {
	case board.Todo:
		return Todo
	case board.AwaitingDesignReview:
		return DesignComplete
	case board.Starting, board.Running:
		if c.Phase == board.DesignPhase {
			return Design
		}
		return Work
	case board.AwaitingResultReview:
		return Review
	case board.Done:
		return Done
	case board.Failed, board.Canceled:
		return FailedOrCanceled
	default:
		return -1
	}
}
