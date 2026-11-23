package board

import (
	"errors"
	"strings"
)

const (
	DesignPhase         = "design"
	ImplementationPhase = "implementation"
)

func (c Card) CanStartDesign() bool {
	return c.manual() && (c.Status == Todo || (c.Status == Failed && c.Phase == DesignPhase))
}

func (c Card) CanReviewDesign() bool {
	return c.manual() && c.Status == AwaitingDesignReview && c.Phase == DesignPhase && c.Design != ""
}

func (c Card) CanReviewResult() bool {
	return c.manual() && c.Phase == ImplementationPhase && c.Status == AwaitingResultReview
}

func (c *Card) ApproveResult() bool {
	if !c.CanReviewResult() {
		return false
	}
	c.Status = Done
	return true
}

// PreparePhase は起動前にフェーズを検証し、差し戻しなら次の実行に渡すレビュー内容を記録する。
func (c *Card) PreparePhase(phase, feedback string) error {
	switch phase {
	case DesignPhase:
		if c.CanReviewDesign() {
			feedback = strings.TrimSpace(feedback)
			if feedback == "" {
				return errors.New("再設計のコメントを入力してください")
			}
			c.Feedback = feedback
		} else if !c.CanStartDesign() {
			return errors.New("このカードは設計を開始できません")
		}
	case ImplementationPhase:
		if c.CanReviewResult() {
			feedback = strings.TrimSpace(feedback)
			if feedback == "" {
				return errors.New("実装の修正内容を入力してください")
			}
			c.ImplementationFeedback, c.PreviousResult = feedback, c.Result
		} else if !c.CanReviewDesign() {
			return errors.New("レビュー待ちの設計案がありません")
		}
	default:
		return errors.New("不明なフェーズです")
	}
	c.Phase = phase
	return nil
}

func (c Card) PhaseLabel() string {
	switch c.Status {
	case Canceled:
		return "キャンセル済み"
	case Failed:
		switch c.Phase {
		case DesignPhase:
			return "設計失敗"
		case ImplementationPhase:
			return "実装失敗"
		}
		return "失敗"
	case AwaitingDesignReview:
		return "設計完了（承認待ち）"
	case AwaitingResultReview:
		return "実装結果の確認待ち"
	}
	switch c.Phase {
	case DesignPhase:
		return "設計中"
	case ImplementationPhase:
		if c.Status == Done {
			return "実装完了"
		}
		return "実装中"
	}
	return ""
}
