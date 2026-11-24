package automation

import (
	"fmt"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/schedule"
)

// TimingInput はフッターで時刻・間隔を編集するときのラベルと初期値を返す。
func TimingInput(r schedule.Rule) (string, string) {
	label := "時刻（HH:MM）"
	if r.TimingKind() == schedule.Every {
		label = "間隔（例 15m）"
	}
	return fmt.Sprintf("#%d %s > ", r.ID, label), r.TimingValue()
}
