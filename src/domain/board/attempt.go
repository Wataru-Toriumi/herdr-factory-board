package board

func (c Card) IsRunningAttempt(attempt int) bool {
	return c.Attempt == attempt && c.Status == Running
}

// FinishAttempt は実行開始時のフェーズに従って結果を記録する。
// キャンセルや再試行の後に届いた結果は破棄する。
func (c *Card) FinishAttempt(attempt int, phase, status, result string) {
	if !c.IsRunningAttempt(attempt) {
		return
	}
	c.Status, c.Result = status, result
	if status != Done {
		return
	}
	switch phase {
	case DesignPhase:
		c.Design, c.Result, c.Status = result, "", AwaitingDesignReview
	case ImplementationPhase:
		c.Status = AwaitingResultReview
	}
}
