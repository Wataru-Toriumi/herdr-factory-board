package board

// QueueAttempt は開始条件の確認後に呼び、ワーカーへ渡す実行番号を返す。
func (c *Card) QueueAttempt() int {
	if c.Status == Todo && c.ReopenCount > 0 {
		// 再オープン前の記録は、新しい作業を開始するまで詳細画面に残す。
		c.Design, c.Feedback, c.ImplementationFeedback, c.PreviousResult = "", "", "", ""
	}
	c.Attempt++
	c.Status, c.Pane, c.Result = Starting, "", ""
	return c.Attempt
}

func (c Card) isStartingAttempt(attempt int) bool {
	return c.Attempt == attempt && c.Status == Starting
}

func (c *Card) MarkAttemptRunning(attempt int, pane string) bool {
	if !c.isStartingAttempt(attempt) {
		return false
	}
	c.Status, c.Pane = Running, pane
	return true
}

func (c *Card) FailAttemptLaunch(attempt int, message string) {
	if c.isStartingAttempt(attempt) {
		c.Status, c.Result = Failed, message
	}
}

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
