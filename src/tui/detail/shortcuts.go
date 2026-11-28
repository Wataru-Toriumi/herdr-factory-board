package detail

import "github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"

type Action int

const (
	CancelCard Action = iota
	ChangeCardAgent
	ChangeProject
	StartDesign
	ApproveResult
	EditFeedback
	StartImplementation
	ReopenCard
	ChangeScheduleAgent
	EditTiming
	EditScheduleSettings
	OpenChat
	ApplyChat
)

type Shortcut struct {
	Key    rune
	Label  string
	Action Action
}

type ShortcutList []Shortcut

func Shortcuts(card *board.Card, includeSchedule bool) ShortcutList {
	shortcuts := cardShortcuts(card)
	if includeSchedule {
		if card == nil {
			shortcuts = append(shortcuts,
				Shortcut{'e', "予定を編集", EditScheduleSettings},
				Shortcut{'g', "担当を変更", ChangeScheduleAgent},
			)
		}
		shortcuts = append(shortcuts, Shortcut{'t', "定期実行の設定変更", EditTiming})
	}
	return shortcuts
}

func (s ShortcutList) Hint() string {
	hint := ""
	for _, shortcut := range s {
		hint += string(shortcut.Key) + " " + shortcut.Label + "   "
	}
	return hint + "↑↓ スクロール   Esc 戻る"
}

func cardShortcuts(c *board.Card) ShortcutList {
	if c == nil {
		return nil
	}
	var shortcuts ShortcutList
	if c.CanChat() {
		shortcuts = append(shortcuts, Shortcut{'c', "会話", OpenChat})
	}
	if c.CanApplyChat() {
		label := "会話で設計を更新"
		if c.CanReviewResult() {
			label = "会話で実装を修正"
		}
		shortcuts = append(shortcuts, Shortcut{'f', label, ApplyChat})
	}
	if c.CanCancel() {
		shortcuts = append(shortcuts, Shortcut{'x', "キャンセル", CancelCard})
	}
	if c.CanConfigure() {
		shortcuts = append(shortcuts, Shortcut{'g', "担当を変更", ChangeCardAgent})
		shortcuts = append(shortcuts, Shortcut{'p', "作業先を変更", ChangeProject})
	}
	switch {
	case c.CanStartDesign():
		return append(shortcuts, Shortcut{'d', "設計から開始", StartDesign})
	case c.CanReviewResult():
		return append(shortcuts,
			Shortcut{'a', "結果を承認して完了", ApproveResult},
			Shortcut{'r', "修正を依頼", EditFeedback},
		)
	case c.CanReviewDesign():
		return append(shortcuts,
			Shortcut{'a', "承認して実装", StartImplementation},
			Shortcut{'r', "コメントして再設計", EditFeedback},
		)
	}
	switch c.ReopenTarget() {
	case board.Todo:
		return append(shortcuts, Shortcut{'u', "未着手に戻す", ReopenCard})
	case board.AwaitingResultReview:
		return append(shortcuts, Shortcut{'u', "承認を取り消す", ReopenCard})
	}
	return shortcuts
}
