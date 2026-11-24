package taskboard

import (
	"github.com/gdamore/tcell/v2"
)

var taskLanes = [LaneCount]struct {
	name  string
	color tcell.Color
}{
	Todo:             {"未着手", tcell.ColorLightSkyBlue},
	Design:           {"設計中", tcell.ColorPlum},
	DesignComplete:   {"設計完了", tcell.ColorLavender},
	Work:             {"進行中", tcell.ColorGold},
	Review:           {"レビュー中", tcell.ColorLightCyan},
	Done:             {"完了", tcell.ColorLightGreen},
	FailedOrCanceled: {"失敗・中止", tcell.ColorLightCoral},
}
