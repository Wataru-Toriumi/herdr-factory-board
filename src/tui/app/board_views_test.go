package app

import (
	"strings"
	"testing"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/schedule"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/store"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/tui/taskboard"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestCompactBoardFollowsLaneSelection(t *testing.T) {
	u := newBoardUI(store.Store{Dir: t.TempDir()}, "")
	u.tasks.Resize(40)
	cards := []board.Card{{ID: 1, Status: board.Todo}}
	u.tasks.Render(cards, u.projects, u.workDir)
	check := func(lane int) {
		t.Helper()
		if u.tasks.GetItemCount() != 1 || u.tasks.GetItem(0) != u.tasks.List(lane) || u.app.GetFocus() != u.tasks.List(lane) {
			t.Fatalf("表示列とフォーカスが %s に揃っていません", u.tasks.List(lane).GetTitle())
		}
	}
	u.key(tcell.NewEventKey(tcell.KeyLeft, 0, 0))
	check(taskboard.FailedOrCanceled)
	u.key(tcell.NewEventKey(tcell.KeyRight, 0, 0))
	check(taskboard.Todo)
	cards[0].Status = board.Running
	u.tasks.Render(cards, u.projects, u.workDir)
	check(taskboard.Work)
}

func TestRefreshPreservesActiveView(t *testing.T) {
	for _, tc := range []struct {
		name string
		key  rune
		page string
	}{
		{"コマンド入力", ':', "tasks"},
		{"詳細", '?', "detail"},
		{"作業先の選択", 'p', "directory"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := store.Store{Dir: t.TempDir()}
			u := newBoardUI(s, "")
			u.workDir = s.Dir
			cards := []board.Card{{ID: 1, Status: board.Todo, Dir: s.Dir}}
			u.tasks.Render(cards, u.projects, u.workDir)
			u.key(tcell.NewEventKey(tcell.KeyRune, tc.key, 0))
			page, view := u.pages.GetFrontPage()
			if page != tc.page {
				t.Fatalf("表示中のページ = %q, 期待値 = %q", page, tc.page)
			}
			if tc.key == ':' {
				view = u.input
			}
			cards[0].Status = board.Running
			u.tasks.Render(cards, u.projects, u.workDir)
			if u.app.GetFocus() != view {
				t.Fatal("カードの列移動で操作中の画面からフォーカスが外れました")
			}
			u.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
			if u.app.GetFocus() != u.tasks.List(taskboard.Work) {
				t.Fatal("Esc で進行中のカードに戻れませんでした")
			}
		})
	}
}

func TestDetailKeysWithoutSchedule(t *testing.T) {
	u := newBoardUI(store.Store{Dir: t.TempDir()}, "")
	for _, tc := range []struct {
		name            string
		card            *board.Card
		consumeAgentKey bool
	}{
		{"解除済みの予定", nil, true},
		{"解除済みの予定から作成したカード", &board.Card{Status: board.Running}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			u.openDetail(tc.name, "", 1, tc.card)
			if hint := u.hint.GetText(false); strings.Contains(hint, "g 担当を変更") || strings.Contains(hint, "t 定期実行の設定変更") {
				t.Fatalf("解除後の操作案内 = %q", hint)
			}
			for key, consume := range map[rune]bool{'g': tc.consumeAgentKey, 't': true, 'j': false} {
				event := tcell.NewEventKey(tcell.KeyRune, key, 0)
				if got := u.key(event); (consume && got != nil) || (!consume && got != event) {
					t.Errorf("キー %q の受け渡しが違います", key)
				}
			}
		})
	}
}

func TestDetailRefreshShowsUpdatedCardAndSchedule(t *testing.T) {
	s := store.Store{Dir: t.TempDir()}
	if err := s.Edit(func(b *board.State) error {
		b.Schedules = []schedule.Rule{{ID: 1, Time: "08:00"}}
		b.Cards = []board.Card{{ID: 2, ScheduleID: 1, Status: board.Running, Prompt: "確認"}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	u := newBoardUI(s, "true")
	u.execute("result 2")
	if err := s.Edit(func(b *board.State) error {
		b.Schedules[0].Time = "10:30"
		b.Cards[0].Status, b.Cards[0].Result = board.Done, "確認完了"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	u.refresh()
	if text := u.detail.GetText(false); !strings.Contains(text, "毎日 10:30") || !strings.Contains(text, "確認完了") {
		t.Fatalf("更新後の予定と結果: %s", text)
	}
	if hint := u.hint.GetText(false); strings.Contains(hint, "x キャンセル") || !strings.Contains(hint, "t 定期実行の設定変更") {
		t.Fatalf("完了した定期カードの操作案内: %s", hint)
	}
}

func TestTabNavigationRestoresPageHintAndFocus(t *testing.T) {
	u := newBoardUI(store.Store{Dir: t.TempDir()}, "")
	u.key(tcell.NewEventKey(tcell.KeyRight, 0, 0))
	for _, tc := range []struct {
		key               tcell.Key
		page, label, hint string
		focus             tview.Primitive
	}{
		{tcell.KeyTab, "agents", "エージェント", "Enter ペインへ", u.agents},
		{tcell.KeyTab, "reviews", "レビュー", "Enter GitHubで開く", u.reviews},
		{tcell.KeyTab, "automation", "自動化", "i 間隔", u.schedules},
		{tcell.KeyTab, "tasks", "タスク", "v 詳細", u.tasks.List(taskboard.Design)},
		{tcell.KeyBacktab, "automation", "自動化", "i 間隔", u.schedules},
	} {
		u.key(tcell.NewEventKey(tc.key, 0, 0))
		if u.activePage() != tc.page || u.app.GetFocus() != tc.focus {
			t.Fatalf("%s の表示先またはフォーカスが違います: %s", tc.label, u.activePage())
		}
		if !strings.Contains(u.header.GetText(false), "[black:aqua:b] "+tc.label+" [-:-:-]") || !strings.Contains(u.hint.GetText(false), tc.hint) {
			t.Fatalf("%s の見出しまたは操作案内が違います: %s, %s", tc.label, u.header.GetText(false), u.hint.GetText(false))
		}
	}
}
