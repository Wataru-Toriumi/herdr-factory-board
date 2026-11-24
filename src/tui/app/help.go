package app

import (
	"fmt"
	"strings"
)

func (u *boardUI) help() {
	var text strings.Builder
	fmt.Fprintln(&text, "タスクを選んで v で詳細を表示。Enter で開始・担当ペインへ移動・結果表示。")
	fmt.Fprintln(&text, "カードの詳細で x を押すと、そのタスクをキャンセルします。定期実行の次回以降の予定は継続します。")
	fmt.Fprintln(&text, "手動タスクの詳細で d から設計。設計案を確認し、a で承認・実装、r でコメントして再設計。実装後はレビュー中のカードで結果を確認し、a で完了。")
	fmt.Fprintln(&text, "実装レビュー中は r で修正を依頼。承認後は u でレビュー中に戻せます。キャンセル済みの手動タスクは u で未着手に戻せます。")
	fmt.Fprintln(&text, "一覧の p で表示を絞り込みます。n で新規タスクのフォームを開きます。Ctrl+P で作業先を選択、本文は Enter で改行、Ctrl+S で保存。未着手の手動タスクは詳細の p で作業先を変更できます。")
	fmt.Fprintln(&text, "新規タスク・予定の入力中は Ctrl+G、未着手の手動カード・予定の詳細では g で担当を切り替えます。")
	fmt.Fprintln(&text, "コマンドはそのまま入力し、Enter で確定。Esc でキャンセル。")
	fmt.Fprintln(&text, "dir 絶対パス / add 作業内容 / daily HH:MM 作業内容 / every 15m 作業内容")
	fmt.Fprintln(&text, "start 番号 / open 番号 / result 番号 / recover 番号")
	fmt.Fprintln(&text, "scheduler / remove 定期番号")
	fmt.Fprintln(&text, "自動化タブでは n で毎日、i で間隔指定の登録フォームを開きます。t で時刻・間隔の変更、Enter で詳細、x で解除。")
	fmt.Fprintln(&text, "定期実行の詳細では e で設定を編集。実行内容は Enter で改行、Ctrl+S で保存します。変更は次に作成されるカードから反映されます。")
	fmt.Fprintln(&text, "\nワークスペース")
	for _, w := range u.spaces {
		label := w.Label
		if label == "" {
			label = w.ID
		}
		fmt.Fprintf(&text, "%s  %s\n", label, w.AgentStatus)
	}
	u.openDetail("操作方法", text.String(), 0, nil)
}
