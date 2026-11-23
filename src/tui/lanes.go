package tui

import (
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/gdamore/tcell/v2"
)

const (
	todoLane = iota
	designLane
	designCompleteLane
	workLane
	reviewLane
	doneLane
	failedOrCanceledLane
)

var taskLanes = [...]struct {
	name  string
	color tcell.Color
}{
	todoLane:             {"未着手", tcell.ColorLightSkyBlue},
	designLane:           {"設計中", tcell.ColorPlum},
	designCompleteLane:   {"設計完了", tcell.ColorLavender},
	workLane:             {"進行中", tcell.ColorGold},
	reviewLane:           {"レビュー中", tcell.ColorLightCyan},
	doneLane:             {"完了", tcell.ColorLightGreen},
	failedOrCanceledLane: {"失敗・中止", tcell.ColorLightCoral},
}

func cardLane(c board.Card) int {
	switch c.Status {
	case board.Todo:
		return todoLane
	case board.AwaitingDesignReview:
		return designCompleteLane
	case board.Starting, board.Running:
		if c.Phase == board.DesignPhase {
			return designLane
		}
		return workLane
	case board.AwaitingResultReview:
		return reviewLane
	case board.Done:
		return doneLane
	case board.Failed, board.Canceled:
		return failedOrCanceledLane
	default:
		return -1
	}
}
