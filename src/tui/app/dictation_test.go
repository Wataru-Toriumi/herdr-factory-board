package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/schedule"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/store"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestDictationForms(t *testing.T) {
	for _, tc := range []struct {
		name   string
		action string
		script string
		want   string
	}{
		{"新規タスク", "stop", "printf 'ready\\n'; read -r cmd && [ \"$cmd\" = stop ] && printf '日本語'", "前日本語後"},
		{"チケットの会話", "stop", "printf 'ready\\n'; read -r cmd && [ \"$cmd\" = stop ] && printf '日本語'", "前日本語後"},
		{"予定の編集", "stop", "printf 'ready\\n'; read -r cmd && [ \"$cmd\" = stop ] && printf '日本語'", "前日本語後"},
		{"取消", "cancel", "printf 'ready\\n'; read -r cmd", "前文後"},
		{"許可エラー", "error", "echo 'マイクの許可が必要です' >&2; exit 1", "前文後"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			u := newBoardUI(store.Store{Dir: t.TempDir()}, "")
			u.voiceHelper = filepath.Join(t.TempDir(), "dictation")
			if err := os.WriteFile(u.voiceHelper, []byte("#!/bin/sh\n"+tc.script+"\n"), 0700); err != nil {
				t.Fatal(err)
			}
			var field *tview.TextArea
			page := "task-form"
			switch tc.name {
			case "チケットの会話":
				u.openCardDetail(board.Card{ID: 1, Status: board.Todo})
				u.openChat()
				field = u.chatForm.Message.SetText("前文後", true)
				page = "chat"
			case "予定の編集":
				u.openScheduleForm(schedule.Rule{ID: 1, Dir: "/tmp", Prompt: "前文後", Time: "08:00"}, schedule.Daily)
				field = u.scheduleForm.Prompt
				page = "schedule-form"
			default:
				u.newTaskForm()
				field = u.taskForm.Prompt.SetText("前文後", true)
			}
			field.Select(len("前"), len("前文"))
			press := func(k tcell.Key) { u.key(tcell.NewEventKey(k, 0, 0)) }
			press(tcell.KeyCtrlR)
			if u.activePage() != "dictation" {
				t.Fatalf("録音画面を開けません: %s", u.notice.GetText(false))
			}
			t.Cleanup(func() {
				if u.voice != nil {
					u.voice.session.Cancel()
				}
			})
			waitFor := func(done func() bool) {
				t.Helper()
				until := time.Now().Add(3 * time.Second)
				for !done() {
					u.updateVoiceInput()
					if time.Now().After(until) {
						t.Fatal("音声入力の応答がありません")
					}
					time.Sleep(time.Millisecond)
				}
			}
			if tc.action != "error" {
				waitFor(func() bool { return u.voice != nil && u.voice.ready })
				if tc.action == "cancel" {
					press(tcell.KeyEscape)
				} else {
					press(tcell.KeyCtrlR)
				}
			}
			waitFor(func() bool { return u.voice == nil })
			if u.activePage() != page || field.GetText() != tc.want || u.app.GetFocus() != field {
				t.Fatalf("復帰先=%s, 本文=%q", u.activePage(), field.GetText())
			}
			if tc.action == "error" && !strings.Contains(u.notice.GetText(false), "マイクの許可") {
				t.Fatal("許可エラーが表示されません")
			}
		})
	}
}
