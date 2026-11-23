package worker

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// 担当ペインには経過を表示し、カードには最後の result イベントの本文を保存する。
func readClaudeStream(stdout io.Reader, pane io.Writer) (string, error) {
	dec := json.NewDecoder(stdout)
	progress := claudeProgress{pane: pane}
	var result string
	var finished bool
	var resultErr error
	for {
		var event struct {
			Type, Subtype, Result string
			IsError               bool `json:"is_error"`
			Errors                []string
			Message               json.RawMessage
		}
		err := dec.Decode(&event)
		if err == io.EOF {
			if finished {
				return result, resultErr
			}
			return progress.lastText, errors.New("Claude Code の終了結果を取得できませんでした")
		}
		if err != nil {
			return progress.lastText, fmt.Errorf("Claude Code の出力を読み取れません: %w", err)
		}
		switch event.Type {
		case "assistant":
			if err := progress.showMessage(event.Message); err != nil {
				return progress.lastText, err
			}
		case "result":
			finished, result = true, event.Result
			if event.IsError || event.Subtype != "success" {
				resultErr = fmt.Errorf("Claude Code の実行に失敗しました (%s)", event.Subtype)
				result = strings.TrimSpace(strings.Join(append([]string{result}, event.Errors...), "\n"))
			}
		}
	}
}

// ツールの通知だけが続いても、中断時に返す最後の本文を保持する。
type claudeProgress struct {
	pane     io.Writer
	lastText string
}

func (p *claudeProgress) showMessage(raw json.RawMessage) error {
	var message struct {
		Content []struct{ Type, Text, Name string }
	}
	if err := json.Unmarshal(raw, &message); err != nil {
		return fmt.Errorf("Claude Code のメッセージを読み取れません: %w", err)
	}
	for _, block := range message.Content {
		switch block.Type {
		case "text":
			p.lastText = block.Text
			fmt.Fprintln(p.pane, block.Text)
		case "tool_use":
			fmt.Fprintln(p.pane, "ツール:", block.Name)
		}
	}
	return nil
}
