package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/application/command"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/schedule"
	"github.com/gdamore/tcell/v2"
)

func (u *boardUI) help() {
	var text strings.Builder
	fmt.Fprintln(&text, "タスクを選んで v で詳細を表示。Enter で開始・担当ペインへ移動・結果表示。")
	fmt.Fprintln(&text, "カードの詳細で x を押すと、そのタスクをキャンセルします。定期実行の次回以降の予定は継続します。")
	fmt.Fprintln(&text, "手動タスクの詳細で d から設計。設計案を確認し、a で承認・実装、r でコメントして再設計。実装後はレビュー中のカードで結果を確認し、a で完了。")
	fmt.Fprintln(&text, "実装レビュー中は r で修正を依頼。承認後は u でレビュー中に戻せます。キャンセル済みの手動タスクは u で未着手に戻せます。")
	fmt.Fprintln(&text, "一覧の p で表示を絞り込みます。n でタスクごとの作業先を選択。未着手の手動タスクは詳細の p で作業先を変更できます。")
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

func (u *boardUI) execute(line string) {
	u.executeAt(command.Parse(line), "")
}

func (u *boardUI) executeAt(cmd command.Command, dir string) {
	u.notice.SetText("")
	if dir == "" {
		run := func(selected string) { u.runCommand(cmd, selected) }
		switch cmd.Name {
		case "add":
			if cmd.Arg != "" {
				u.pickProject(u.workDir, false, run)
				return
			}
		case schedule.Daily, schedule.Every:
			u.selectScheduleProject(run)
			return
		}
		dir = u.workDir
	}
	u.runCommand(cmd, dir)
}

func (u *boardUI) runCommand(cmd command.Command, dir string) {
	c, err := cmd.Run(u.store, u.herdr, dir)
	if err == nil && cmd.Name == "dir" {
		u.workDir = filepath.Clean(cmd.Arg)
	}
	u.refresh()
	if err != nil {
		u.notice.SetText(err.Error())
	} else if cmd.Name == "add" && u.workDir != "" && dir != u.workDir {
		u.notice.SetText("チケットを追加しました。作業先: " + dir)
	} else if c != nil {
		u.openCardDetail(*c)
	}
}

func (u *boardUI) activate() {
	switch u.tab {
	case automationTab:
		if r, ok := u.schedules.selected(); ok {
			u.openDetail(fmt.Sprintf("定期実行 #%d", r.ID), "", r.ID, nil)
		}
		return
	case agentTab:
		if row, ok := u.agents.selected(); ok {
			if err := u.focusAgent(row); err != nil {
				u.notice.SetText(err.Error())
			}
		}
		return
	}
	c, ok := u.selectedCard()
	if !ok {
		return
	}
	action := "result"
	switch c.Status {
	case board.Todo:
		action = "start"
	case board.Running:
		action = "open"
	}
	u.execute(fmt.Sprintf("%s %d", action, c.ID))
}

func (u *boardUI) key(event *tcell.EventKey) *tcell.EventKey {
	key := event.Key()
	switch u.activePage() {
	case "directory":
		if key == tcell.KeyEscape {
			u.closeDirectory()
			u.refresh()
			return nil
		}
		return event
	case "schedule-form":
		return u.scheduleFormKey(event)
	case "input":
		return u.inputKey(event)
	case "detail":
		if key == tcell.KeyRune && u.detailKey(event.Rune()) {
			return nil
		}
		if key == tcell.KeyEscape {
			u.switchView()
			return nil
		}
		return event
	}
	switch key {
	case tcell.KeyTab, tcell.KeyBacktab:
		step := 1
		if key == tcell.KeyBacktab {
			step = -1
		}
		u.tab = (u.tab + step + len(u.tabs)) % len(u.tabs)
		u.notice.SetText("")
		u.switchView()
	case tcell.KeyLeft, tcell.KeyRight:
		if u.tab == taskTab {
			direction := 1
			if key == tcell.KeyLeft {
				direction = -1
			}
			u.selectLane((u.lane + direction + len(taskLanes)) % len(taskLanes))
		}
	case tcell.KeyEnter:
		u.activate()
	case tcell.KeyRune:
		if u.tab == taskTab && event.Rune() == 'v' {
			if c, ok := u.selectedCard(); ok {
				u.execute(fmt.Sprintf("result %d", c.ID))
			}
			return nil
		}
		if u.tab == automationTab {
			switch event.Rune() {
			case 't':
				if r, ok := u.schedules.selected(); ok {
					u.editScheduleTiming(r)
				}
				return nil
			case 'x':
				if r, ok := u.schedules.selected(); ok {
					u.editCommand(fmt.Sprintf("remove %d", r.ID))
				}
				return nil
			}
		}
		switch event.Rune() {
		case 'n':
			if u.tab == automationTab {
				u.newScheduleForm(schedule.Daily)
			} else {
				u.pickProject(u.workDir, false, func(dir string) { u.editForProject("add ", dir) })
			}
		case 'i':
			if u.tab == automationTab {
				u.newScheduleForm(schedule.Every)
			} else {
				u.editCommand("i")
			}
		case 'p':
			u.pickProject(u.workDir, true, func(dir string) {
				u.workDir = dir
				u.refresh()
			})
		case 'q', 'Q':
			u.app.Stop()
		case ':':
			u.editCommand("")
		case '?':
			u.help()
		default:
			u.editCommand(string(event.Rune()))
		}
	default:
		return event
	}
	return nil
}
