package main

import (
	"errors"
	"fmt"
	"strings"
)

const (
	designPhase         = "design"
	implementationPhase = "implementation"
)

func (c card) canStartDesign() bool {
	return c.Source == "" && c.ScheduleID == 0 && (c.Status == todo || (c.Status == failed && c.Phase == designPhase))
}

func (c card) canReviewDesign() bool {
	return c.Source == "" && c.ScheduleID == 0 && c.Status == awaitingReview && c.Phase == designPhase && c.Design != ""
}

func (c card) phaseLabel() string {
	switch c.Status {
	case failed:
		switch c.Phase {
		case designPhase:
			return "設計失敗"
		case implementationPhase:
			return "実装失敗"
		}
		return "失敗"
	case awaitingReview:
		return "設計完了（承認待ち）"
	case awaitingResultReview:
		return "実装結果の確認待ち"
	}
	switch c.Phase {
	case designPhase:
		return "設計中"
	case implementationPhase:
		if c.Status == done {
			return "実装完了"
		}
		return "実装中"
	}
	return ""
}

func approveResult(s store, seen card) error {
	return s.editCard(seen.ID, func(c *card) error {
		if c.Status != awaitingResultReview || c.Attempt != seen.Attempt {
			return errors.New("カードが更新されました。最新の実行結果を確認してください")
		}
		c.Status = done
		return nil
	})
}

// レビューで表示した版と保存済みのカードをロック内で照合し、別の設計案の承認を防ぐ。
func launchFromDetail(s store, herdr string, seen card, phase, feedback string) error {
	return launchCard(s, herdr, seen.ID, func(c *card) error {
		if c.Attempt != seen.Attempt || c.Status != seen.Status {
			return errors.New("カードが更新されました。最新の詳細を確認してください")
		}
		switch phase {
		case designPhase:
			if c.canReviewDesign() {
				feedback = strings.TrimSpace(feedback)
				if feedback == "" {
					return errors.New("再設計のコメントを入力してください")
				}
				c.Feedback = feedback
			} else if !c.canStartDesign() {
				return errors.New("このカードは設計を開始できません")
			}
		case implementationPhase:
			if !c.canReviewDesign() {
				return errors.New("レビュー待ちの設計案がありません")
			}
		default:
			return errors.New("不明なフェーズです")
		}
		c.Phase = phase
		return nil
	})
}

func (c card) workerPrompt() string {
	switch c.Phase {
	case designPhase:
		prompt := "リポジトリを調査して設計案を作成してください。実装やファイル変更は行わず、変更方針、受け入れ条件、テスト方針、未決事項を最終メッセージにまとめてください。\n\n元の依頼\n" + c.Prompt
		if c.Design != "" {
			prompt += "\n\n前回の設計案\n" + c.Design + "\n\nレビューコメント\n" + c.Feedback
		}
		return prompt
	case implementationPhase:
		return fmt.Sprintf("承認された設計案に沿って実装し、テスト結果を報告してください。\n\n元の依頼\n%s\n\n承認済みの設計案\n%s", c.Prompt, c.Design)
	default:
		return c.Prompt
	}
}
