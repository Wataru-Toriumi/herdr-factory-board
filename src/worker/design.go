package worker

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/store"
)

func ApproveResult(s store.Store, seen board.Card) error {
	return s.EditCard(seen.ID, func(c *board.Card) error {
		if c.Status != board.AwaitingResultReview || c.Attempt != seen.Attempt {
			return errors.New("カードが更新されました。最新の実行結果を確認してください")
		}
		c.Status = board.Done
		return nil
	})
}

// レビューで表示した版と保存済みのカードをロック内で照合し、別の設計案の承認を防ぐ。
func StartPhase(s store.Store, herdr string, seen board.Card, phase, feedback string) error {
	return launchCard(s, herdr, seen.ID, func(c *board.Card) error {
		if c.Attempt != seen.Attempt || c.Status != seen.Status || c.Dir != seen.Dir {
			return errors.New("カードが更新されました。最新の詳細を確認してください")
		}
		switch phase {
		case board.DesignPhase:
			if c.CanReviewDesign() {
				feedback = strings.TrimSpace(feedback)
				if feedback == "" {
					return errors.New("再設計のコメントを入力してください")
				}
				c.Feedback = feedback
			} else if !c.CanStartDesign() {
				return errors.New("このカードは設計を開始できません")
			}
		case board.ImplementationPhase:
			if !c.CanReviewDesign() {
				return errors.New("レビュー待ちの設計案がありません")
			}
		default:
			return errors.New("不明なフェーズです")
		}
		c.Phase = phase
		return nil
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
		return fmt.Sprintf("承認された設計案に沿って実装し、テスト結果を報告してください。\n\n元の依頼\n%s\n\n承認済みの設計案\n%s", c.Prompt, c.Design)
	default:
		return c.Prompt
	}
}
