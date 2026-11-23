package worker

import (
	"errors"
	"fmt"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/store"
)

func ApproveResult(s store.Store, seen board.Card) error {
	return s.EditCard(seen.ID, func(c *board.Card) error {
		if *c != seen || !c.CanReviewResult() {
			return errors.New("カードが更新されました。最新の実行結果を確認してください")
		}
		c.Status = board.Done
		return nil
	})
}

func Reopen(s store.Store, seen board.Card) error {
	return s.EditCard(seen.ID, func(c *board.Card) error {
		if *c != seen || !c.Reopen() {
			return errors.New("カードを戻せません。最新の状態を確認してください")
		}
		return nil
	})
}

// レビューで表示した版と保存済みのカードをロック内で照合し、別の設計案の承認を防ぐ。
func StartPhase(s store.Store, herdr string, seen board.Card, phase, feedback string) error {
	return launchCard(s, herdr, seen.ID, func(c *board.Card) error {
		if *c != seen {
			return errors.New("カードが更新されました。最新の詳細を確認してください")
		}
		return c.PreparePhase(phase, feedback)
	})
}

func workerPrompt(c board.Card) string {
	switch c.Phase {
	case board.DesignPhase:
		prompt := "リポジトリを調査して設計案を作成してください。実装やファイル変更は行わず、変更方針、受け入れ条件、テスト方針、未決事項を最終メッセージにまとめてください。\n\n元の依頼\n" + c.Prompt
		if c.Design != "" {
			prompt += "\n\n前回の設計案\n" + c.Design + "\n\nレビューコメント\n" + c.Feedback
		}
		return prompt
	case board.ImplementationPhase:
		prompt := fmt.Sprintf("承認された設計案に沿って実装し、テスト結果を報告してください。\n\n元の依頼\n%s\n\n承認済みの設計案\n%s", c.Prompt, c.Design)
		if c.ImplementationFeedback != "" {
			prompt += "\n\n前回の実装結果\n" + c.PreviousResult + "\n\n以下のレビュー指摘に対応してください\n" + c.ImplementationFeedback
		}
		return prompt
	default:
		return c.Prompt
	}
}
