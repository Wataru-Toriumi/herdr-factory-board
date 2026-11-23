package tui

import (
	"github.com/Wataru-Toriumi/herdr-factory-board/src/tui/taskboard"
	"github.com/gdamore/tcell/v2"
)

var taskLanes = [taskboard.LaneCount]struct {
	name  string
	color tcell.Color
}{
	taskboard.Todo:             {"未着手", tcell.ColorLightSkyBlue},
	taskboard.Design:           {"設計中", tcell.ColorPlum},
	taskboard.DesignComplete:   {"設計完了", tcell.ColorLavender},
	taskboard.Work:             {"進行中", tcell.ColorGold},
	taskboard.Review:           {"レビュー中", tcell.ColorLightCyan},
	taskboard.Done:             {"完了", tcell.ColorLightGreen},
	taskboard.FailedOrCanceled: {"失敗・中止", tcell.ColorLightCoral},
}
