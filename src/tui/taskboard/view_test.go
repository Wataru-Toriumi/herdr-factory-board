package taskboard

import (
	"strings"
	"testing"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestCardSelectionFollowsWorkerAcrossColumns(t *testing.T) {
	view := NewView(Manual, nil, func(tview.Primitive) {})
	cards := []board.Card{{ID: 1, Status: board.Todo}, {ID: 2, Status: board.Todo}, {ID: 3, Status: board.Done}}
	view.Render(cards, nil, "")
	view.List(0).SetCurrentItem(1)
	for _, status := range []string{board.Starting, board.Running, board.Done} {
		cards[1].Status = status
		view.Render(cards, nil, "")
		selected, ok := view.SelectedCard()
		if !ok || selected.ID != 2 || selected.Status != status {
			t.Fatalf("更新後の選択: %+v, %v", selected, ok)
		}
	}
}

func TestTaskLabelsRefreshWhenOnlyProjectNamesChange(t *testing.T) {
	view := NewView(Manual, nil, func(tview.Primitive) {})
	projects := []string{"/backend/api"}
	cards := []board.Card{{ID: 1, Status: board.Todo, Dir: projects[0]}}
	view.Render(cards, projects, "")
	list := view.List(Todo)
	if _, label := list.GetItemText(0); label != "#1  api" {
		t.Fatalf("登録時の表示 = %q", label)
	}

	projects = append(projects, "/frontend/api")
	view.Render(cards, projects, "")
	if _, label := list.GetItemText(0); label != "#1  /backend/api" {
		t.Fatalf("同名プロジェクト追加後の表示 = %q", label)
	}
}

func TestFailedAndCanceledCardsShareLane(t *testing.T) {
	view := NewView(Manual, nil, func(tview.Primitive) {})
	view.Render([]board.Card{{ID: 1, Status: board.Failed}, {ID: 2, Status: board.Canceled}}, nil, "")
	list := view.List(FailedOrCanceled)
	if list.GetItemCount() != 2 {
		t.Fatalf("失敗・中止のカード数 = %d", list.GetItemCount())
	}
	for i, want := range []string{"失敗", "キャンセル済み"} {
		_, label := list.GetItemText(i)
		if !strings.HasPrefix(label, want) {
			t.Fatalf("カードの状態表示 = %q, 期待値 = %q", label, want)
		}
	}
}

func TestTaskLaneTitlesAndSelectionColors(t *testing.T) {
	view := NewView(Manual, nil, func(tview.Primitive) {})
	view.Render(nil, nil, "")
	for _, want := range []struct {
		title string
		color tcell.Color
	}{
		{" 未着手 0 ", tcell.ColorLightSkyBlue},
		{" 設計中 0 ", tcell.ColorPlum},
		{" 設計完了 0 ", tcell.ColorLavender},
		{" 進行中 0 ", tcell.ColorGold},
		{" レビュー中 0 ", tcell.ColorLightCyan},
		{" 完了 0 ", tcell.ColorLightGreen},
		{" 失敗・中止 0 ", tcell.ColorLightCoral},
	} {
		list := view.List(view.Lane())
		if list.GetTitle() != want.title || list.GetBorderColor() != want.color {
			t.Fatalf("選択中の列 = %q (%v), 期待値 = %q (%v)", list.GetTitle(), list.GetBorderColor(), want.title, want.color)
		}
		view.HandleKey(tcell.NewEventKey(tcell.KeyRight, 0, 0))
	}
	if view.Lane() != Todo {
		t.Fatal("最後の列から未着手に戻れませんでした")
	}
}

func TestRunHistoryColumnsAndActions(t *testing.T) {
	var action string
	v := NewView(RunHistory, func(command string) { action = command }, func(tview.Primitive) {})
	cards := []board.Card{
		{ID: 1, Dir: "/api", Status: board.Done, Source: "定期 #7 2026-11-23 10:00"},
		{ID: 2, Dir: "/api", Status: board.Done},
		{ID: 3, Dir: "/api", Status: board.Done, ScheduleID: 7},
		{ID: 4, Dir: "/web", Status: board.Done, ScheduleID: 8},
		{ID: 5, Dir: "/api", Status: board.Todo, ScheduleID: 7},
	}
	v.Render(cards, nil, "/api")
	if v.GetItemCount() != 3 || v.Lane() != Work || v.List(Work).GetItemCount() != 1 {
		t.Fatal("履歴ボードの進行中に起動待ちの実行がありません")
	}
	done := v.List(Done).Rows()
	if len(done) != 2 || done[0].Card.ID != 3 || done[1].Card.ID != 1 {
		t.Fatalf("完了した実行の順序と絞り込み: %+v", done)
	}
	enter := tcell.NewEventKey(tcell.KeyEnter, 0, 0)
	v.HandleKey(enter)
	if action != "result 5" {
		t.Fatalf("起動待ちの実行を開く操作 = %q", action)
	}
	cards[4].Status = board.Running
	v.Render(cards, nil, "/api")
	v.HandleKey(enter)
	if action != "open 5" {
		t.Fatalf("担当ペインを開く操作 = %q", action)
	}
	v.Resize(40)
	cards[4].Status = board.Failed
	v.Render(cards, nil, "/api")
	if v.Lane() != FailedOrCanceled || v.GetItemCount() != 1 || v.GetItem(0) != v.List(FailedOrCanceled) {
		t.Fatal("コンパクト表示で終了した実行に追従できません")
	}
	v.HandleKey(enter)
	if action != "result 5" {
		t.Fatalf("失敗した実行を開く操作 = %q", action)
	}
	v.HandleKey(tcell.NewEventKey(tcell.KeyRight, 0, 0))
	if v.Lane() != Work {
		t.Fatal("履歴の末尾の列から進行中に戻れません")
	}
	manual := Manual.Build(cards, nil, "", Done, 0)
	if rows := manual.Rows[Done]; len(rows) != 1 || rows[0].Card.ID != 2 || len(manual.Rows[FailedOrCanceled]) != 0 {
		t.Fatal("手動ボードに実行履歴が混在しています")
	}
}
