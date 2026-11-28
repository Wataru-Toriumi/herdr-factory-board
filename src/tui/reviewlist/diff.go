package reviewlist

import (
	"strings"

	"github.com/rivo/tview"
)

// DiffText は差分中のタグを文字として表示し、行の種類にだけ色を付ける。
func DiffText(diff string) string {
	if diff == "" {
		return "表示する差分はありません"
	}
	var text strings.Builder
	for diff != "" {
		line, rest, newline := strings.Cut(diff, "\n")
		diff = rest
		color := ""
		switch {
		case strings.HasPrefix(line, "diff --git "):
			color = "[yellow]"
		case strings.HasPrefix(line, "@@"):
			color = "[aqua]"
		case strings.HasPrefix(line, "+"):
			color = "[lightgreen]"
		case strings.HasPrefix(line, "-"):
			color = "[lightcoral]"
		}
		text.WriteString(color)
		text.WriteString(tview.Escape(line))
		if color != "" {
			text.WriteString("[-]")
		}
		if newline {
			text.WriteByte('\n')
		}
	}
	return text.String()
}
