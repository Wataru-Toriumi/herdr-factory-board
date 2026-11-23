package tui

import (
	"github.com/Wataru-Toriumi/herdr-factory-board/src/application/scheduler"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/schedule"
)

type detailShortcut struct {
	key   rune
	label string
	run   func()
}

func (u *boardUI) detailShortcuts(includeSchedule bool) []detailShortcut {
	shortcuts := u.cardShortcuts()
	if !includeSchedule {
		return shortcuts
	}
	if u.detailCard == nil {
		shortcuts = append(shortcuts, detailShortcut{'g', "担当を変更", u.scheduleAction(func(r schedule.Rule) {
			u.finishDetailAction(scheduler.ChangeAgent(u.store, r.ID, r.Agent, nextAgent(r.Agent)))
		})})
	}
	return append(shortcuts, detailShortcut{'t', "定期実行の設定変更", u.scheduleAction(u.editScheduleTiming)})
}

func (u *boardUI) scheduleAction(action func(schedule.Rule)) func() {
	return func() {
		if r, ok := u.schedules.byID(u.detailScheduleID); ok {
			action(r)
		}
	}
}

func (u *boardUI) setDetailHint(hasSchedule bool) {
	hint := ""
	for _, shortcut := range u.detailShortcuts(hasSchedule) {
		hint += string(shortcut.key) + " " + shortcut.label + "   "
	}
	u.hint.SetText(hint + "↑↓ スクロール   Esc 戻る")
}

func (u *boardUI) detailKey(key rune) bool {
	// 予定が解除されても、定期実行の操作キーは本文のスクロールに渡さない。
	for _, shortcut := range u.detailShortcuts(true) {
		if key == shortcut.key {
			shortcut.run()
			return true
		}
	}
	return false
}
