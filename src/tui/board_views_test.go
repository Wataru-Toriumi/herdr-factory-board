package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/application/command"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/schedule"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/store"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/tui/taskboard"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestCardSelectionFollowsWorkerAcrossColumns(t *testing.T) {
	u := newBoardUI(store.Store{Dir: t.TempDir()}, "")
	cards := []board.Card{{ID: 1, Status: board.Todo}, {ID: 2, Status: board.Todo}, {ID: 3, Status: board.Done}}
	u.renderTasks(cards)
	u.lists[0].SetCurrentItem(1)
	for _, status := range []string{board.Starting, board.Running, board.Done} {
		cards[1].Status = status
		u.renderTasks(cards)
		selected, ok := u.selectedCard()
		if !ok || selected.ID != 2 || selected.Status != status {
			t.Fatalf("更新後の選択: %+v, %v", selected, ok)
		}
	}
}

func TestTaskLabelsRefreshWhenOnlyProjectNamesChange(t *testing.T) {
	u := newBoardUI(store.Store{Dir: t.TempDir()}, "")
	u.projects = []string{"/backend/api"}
	cards := []board.Card{{ID: 1, Status: board.Todo, Dir: u.projects[0]}}
	u.renderTasks(cards)
	list := u.lists[taskboard.Todo]
	if _, label := list.GetItemText(0); label != "#1  api" {
		t.Fatalf("登録時の表示 = %q", label)
	}

	u.projects = append(u.projects, "/frontend/api")
	u.renderTasks(cards)
	if _, label := list.GetItemText(0); label != "#1  /backend/api" {
		t.Fatalf("同名プロジェクト追加後の表示 = %q", label)
	}
}

func TestCompactBoardFollowsLaneSelection(t *testing.T) {
	u := newBoardUI(store.Store{Dir: t.TempDir()}, "")
	u.compact = true
	u.layoutColumns()
	cards := []board.Card{{ID: 1, Status: board.Todo}}
	u.renderTasks(cards)
	check := func(lane int) {
		t.Helper()
		if u.columns.GetItemCount() != 1 || u.columns.GetItem(0) != u.lists[lane] || u.app.GetFocus() != u.lists[lane] {
			t.Fatalf("表示列とフォーカスが %s に揃っていません", taskLanes[lane].name)
		}
	}
	u.key(tcell.NewEventKey(tcell.KeyLeft, 0, 0))
	check(taskboard.FailedOrCanceled)
	u.key(tcell.NewEventKey(tcell.KeyRight, 0, 0))
	check(taskboard.Todo)
	cards[0].Status = board.Running
	u.renderTasks(cards)
	check(taskboard.Work)
}

func TestFailedAndCanceledCardsShareLane(t *testing.T) {
	u := newBoardUI(store.Store{Dir: t.TempDir()}, "")
	u.renderTasks([]board.Card{{ID: 1, Status: board.Failed, Source: "定期 #3"}, {ID: 2, Status: board.Canceled}})
	list := u.lists[taskboard.FailedOrCanceled]
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
			u.renderTasks(cards)
			u.key(tcell.NewEventKey(tcell.KeyRune, tc.key, 0))
			page, view := u.pages.GetFrontPage()
			if page != tc.page {
				t.Fatalf("表示中のページ = %q, 期待値 = %q", page, tc.page)
			}
			if tc.key == ':' {
				view = u.input
			}
			cards[0].Status = board.Running
			u.renderTasks(cards)
			if u.app.GetFocus() != view {
				t.Fatal("カードの列移動で操作中の画面からフォーカスが外れました")
			}
			u.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
			if u.app.GetFocus() != u.lists[taskboard.Work] {
				t.Fatal("Esc で進行中のカードに戻れませんでした")
			}
		})
	}
}

func TestAutomationTabManagesDailySchedule(t *testing.T) {
	s := store.Store{Dir: t.TempDir()}
	t.Setenv("HOME", s.Dir)
	u := newBoardUI(s, "true")
	press := func(key tcell.Key) { u.key(tcell.NewEventKey(key, 0, 0)) }
	typeKey := func(ch rune) { u.key(tcell.NewEventKey(tcell.KeyRune, ch, 0)) }

	press(tcell.KeyBacktab)
	typeKey('n')
	page, picker := u.pages.GetFrontPage()
	if page != "directory" {
		t.Fatal("初回の登録で作業先を選択できません")
	}
	picker.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), func(p tview.Primitive) { u.app.SetFocus(p) })
	if u.activePage() != "input" || u.input.GetText() != "daily " {
		t.Fatal("作業先の選択後に定期実行を入力できません")
	}
	u.input.SetText("daily 09:00 テスト")
	press(tcell.KeyEnter)
	b, err := s.Read()
	if err != nil {
		t.Fatal(err)
	}
	r, ok := u.schedules.selected()
	if !ok || r.Prompt != "テスト" || r.Dir != s.Dir || r.Time != "09:00" || len(b.Cards) != 0 {
		t.Fatalf("登録後のボード: %+v", b)
	}

	press(tcell.KeyEnter)
	if u.activePage() != "detail" {
		t.Fatal("予定の詳細を開けません")
	}
	press(tcell.KeyEscape)
	typeKey('x')
	press(tcell.KeyEscape)
	if u.activePage() != "automation" || u.schedules.GetItemCount() != 1 {
		t.Fatal("解除を取り消した後の一覧が不正です")
	}
	typeKey('x')
	press(tcell.KeyEnter)
	b, err = s.Read()
	if err != nil {
		t.Fatal(err)
	}
	if u.schedules.GetItemCount() != 0 || len(b.Schedules) != 0 {
		t.Fatal("確定した予定を解除できません")
	}
	for _, page := range []string{"tasks", "agents", "automation"} {
		press(tcell.KeyTab)
		if u.activePage() != page {
			t.Fatalf("タブ切替後のページ = %q, 期待値 = %q", u.activePage(), page)
		}
	}
}

func TestScheduleSelectionAfterAnotherPaneRemovesRule(t *testing.T) {
	s := store.Store{Dir: t.TempDir()}
	if err := s.Edit(func(b *board.State) error {
		b.Schedules = []schedule.Rule{{ID: 1}, {ID: 2}, {ID: 3}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	u := newBoardUI(s, "true")
	u.refresh()
	u.key(tcell.NewEventKey(tcell.KeyBacktab, 0, 0))
	u.schedules.SetCurrentItem(1)
	if _, err := command.Parse("remove 1").Run(s, "true", s.Dir); err != nil {
		t.Fatal(err)
	}
	u.refresh()
	if count := u.schedules.GetItemCount(); count != 2 {
		t.Fatalf("更新後の予定数 = %d", count)
	}
	u.key(tcell.NewEventKey(tcell.KeyRune, 'x', 0))
	if got := u.input.GetText(); got != "remove 2" {
		t.Fatalf("更新後の解除コマンド = %q", got)
	}
}

func TestAutomationTabIntervalShortcut(t *testing.T) {
	s := store.Store{Dir: t.TempDir()}
	if err := s.RegisterProject(s.Dir); err != nil {
		t.Fatal(err)
	}
	u := newBoardUI(s, "true")
	u.refresh()
	u.key(tcell.NewEventKey(tcell.KeyBacktab, 0, 0))
	u.key(tcell.NewEventKey(tcell.KeyRune, 'i', 0))
	if u.input.GetText() != "every " || u.activePage() != "input" {
		t.Fatal("間隔指定の入力を開始できません")
	}
	u.input.SetText(u.input.GetText() + "5m 変更を確認する")
	u.key(tcell.NewEventKey(tcell.KeyEnter, 0, 0))
	if u.schedules.GetItemCount() != 1 {
		t.Fatal("登録した予定が表示されません")
	}
	text, _ := u.schedules.GetItemText(0)
	if text != "5分おき  変更を確認する" {
		t.Fatalf("一覧の予定: %s", text)
	}
}

func TestEditIntervalFromAutomationTab(t *testing.T) {
	s := store.Store{Dir: t.TempDir()}
	original := time.Now().Add(time.Minute)
	if err := s.Edit(func(b *board.State) error {
		b.NextID = 3
		b.Cards = []board.Card{{ID: 3, Prompt: "履歴", Status: board.Done}}
		b.Schedules = []schedule.Rule{
			{ID: 1, Time: "09:00"},
			{ID: 2, Prompt: "差分の確認", Dir: s.Dir, Interval: &schedule.Interval{Minutes: 1, NextRun: original}},
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	u := newBoardUI(s, "true")
	u.refresh()
	press := func(key tcell.Key) { u.key(tcell.NewEventKey(key, 0, 0)) }
	edit := func() { u.key(tcell.NewEventKey(tcell.KeyRune, 't', 0)) }
	press(tcell.KeyBacktab)
	u.schedules.SetCurrentItem(1)
	edit()
	if u.input.GetText() != "1m" {
		t.Fatal("現在の間隔が入力欄にありません")
	}
	u.input.SetText("30m")
	press(tcell.KeyEscape)
	edit()
	if u.input.GetText() != "1m" {
		t.Fatal("キャンセルした変更が保存されました")
	}
	u.input.SetText("0m")
	press(tcell.KeyEnter)
	if u.activePage() != "input" || u.notice.GetText(false) == "" {
		t.Fatal("入力エラーを修正できません")
	}
	// 編集中に先頭の予定が消えても、保存対象は同じ番号の予定。
	if _, err := command.Parse("remove 1").Run(s, "true", s.Dir); err != nil {
		t.Fatal(err)
	}
	u.refresh()
	before := time.Now()
	u.input.SetText("15m")
	press(tcell.KeyEnter)
	b, err := s.Read()
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Schedules) != 1 || len(b.Cards) != 1 || b.Cards[0].ID != 3 || b.NextID != 3 {
		t.Fatalf("更新後の履歴: %+v", b)
	}
	r := b.Schedules[0]
	if u.activePage() != "automation" || r.ID != 2 || r.Prompt != "差分の確認" || r.Dir != s.Dir || r.Interval.Minutes != 15 {
		t.Fatalf("更新後の予定: %+v", r)
	}
	if r.Interval.NextRun.Before(before.Add(15*time.Minute)) || r.Interval.NextRun.After(time.Now().Add(15*time.Minute)) || r.Due(original) {
		t.Fatalf("次回の実行時刻: %v", r.Interval.NextRun)
	}
	edit()
	press(tcell.KeyEnter)
	same, ok := u.schedules.selected()
	if !ok || !same.Interval.NextRun.Equal(r.Interval.NextRun) {
		t.Fatal("同じ間隔で確定すると実行時刻がずれました")
	}
}

func TestTimingFromTaskDetail(t *testing.T) {
	s := store.Store{Dir: t.TempDir()}
	if err := s.Edit(func(b *board.State) error {
		b.NextID = 5
		b.Schedules = []schedule.Rule{{ID: 1, Time: "08:00"}, {ID: 2, Interval: &schedule.Interval{Minutes: 1}}}
		b.Cards = []board.Card{
			{ID: 3, Status: board.Done, Source: "定期 #2 2026-11-23 10:00", Result: "実行結果"},
			{ID: 4, Status: board.Running, ScheduleID: 2},
			{ID: 5, Status: board.Todo},
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	u := newBoardUI(s, "true")
	u.refresh()
	press := func(k tcell.Key) { u.key(tcell.NewEventKey(k, 0, 0)) }
	key := func(r rune) { u.key(tcell.NewEventKey(tcell.KeyRune, r, 0)) }
	// 一覧の選択とは異なるカードを、番号指定で開く。
	u.execute("result 3")
	key('t')
	if u.activePage() != "input" || u.input.GetText() != "1m" {
		t.Fatal("古いカードの履歴から元の予定を選べません")
	}
	u.input.SetText("15m")
	press(tcell.KeyEnter)
	if text := u.detail.GetText(false); u.activePage() != "detail" || !strings.Contains(text, "15分おき") || !strings.Contains(text, "実行結果") {
		t.Fatalf("保存後のタスク詳細: %s", text)
	}
	press(tcell.KeyEscape)
	for i := taskboard.Todo; i < taskboard.Work; i++ {
		press(tcell.KeyRight)
	}
	key('v')
	key('t')
	u.input.SetText("30m")
	press(tcell.KeyEscape)
	if text := u.detail.GetText(false); u.activePage() != "detail" || !strings.Contains(text, "#4") || !strings.Contains(text, "15分おき") {
		t.Fatalf("取り消し後のタスク詳細: %s", text)
	}
	if _, err := command.Parse("remove 2").Run(s, "true", s.Dir); err != nil {
		t.Fatal(err)
	}
	u.refresh()
	key('t')
	if u.activePage() != "detail" || !strings.Contains(u.detail.GetText(false), "解除済み") {
		t.Fatal("解除済みの予定が編集対象になりました")
	}
	if hint := u.hint.GetText(false); !strings.Contains(hint, "x キャンセル") || strings.Contains(hint, "t 定期実行の設定変更") {
		t.Fatalf("予定解除後の実行中カードの操作案内: %s", hint)
	}
	key('x')
	if u.detailCard.Status != board.Canceled || strings.Contains(u.hint.GetText(false), "x キャンセル") {
		t.Fatal("予定解除後のカードにキャンセルが反映されませんでした")
	}
	u.execute("result 5")
	key('t')
	if u.activePage() != "detail" || strings.Contains(u.hint.GetText(false), "設定変更") {
		t.Fatal("手動タスクに定期実行の編集操作が表示されました")
	}

	press(tcell.KeyEscape)
	press(tcell.KeyBacktab)
	press(tcell.KeyEnter)
	key('t')
	if u.input.GetText() != "08:00" {
		t.Fatal("自動化タブの詳細から時刻を編集できません")
	}
	u.input.SetText(" 10:30 ")
	press(tcell.KeyEnter)
	daily, ok := u.schedules.byID(1)
	if !ok || daily.Time != "10:30" || daily.Interval != nil || u.activePage() != "detail" {
		t.Fatalf("時刻変更後の毎日の予定: %+v", daily)
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
		{tcell.KeyTab, "automation", "自動化", "i 間隔", u.schedules},
		{tcell.KeyTab, "tasks", "タスク", "v 詳細", u.lists[taskboard.Design]},
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

func TestTaskLaneTitlesAndSelectionColors(t *testing.T) {
	u := newBoardUI(store.Store{Dir: t.TempDir()}, "")
	u.renderTasks(nil)
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
		list := u.lists[u.lane]
		if list.GetTitle() != want.title || list.GetBorderColor() != want.color {
			t.Fatalf("選択中の列 = %q (%v), 期待値 = %q (%v)", list.GetTitle(), list.GetBorderColor(), want.title, want.color)
		}
		u.key(tcell.NewEventKey(tcell.KeyRight, 0, 0))
	}
	if u.lane != taskboard.Todo {
		t.Fatal("最後の列から未着手に戻れませんでした")
	}
}

func TestTimingConflictReloadsCurrentSetting(t *testing.T) {
	for _, tc := range []struct {
		name                string
		rule                schedule.Rule
		newValue, nextValue string
	}{
		{"毎日", schedule.Rule{Time: "09:00"}, "10:30", "11:00"},
		{"間隔", schedule.Rule{Interval: &schedule.Interval{Minutes: 15}}, "30m", "45m"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := store.Store{Dir: t.TempDir()}
			if err := s.Edit(func(b *board.State) error { b.AddSchedule(tc.rule); return nil }); err != nil {
				t.Fatal(err)
			}
			left, right := newBoardUI(s, "true"), newBoardUI(s, "true")
			for _, u := range []*boardUI{left, right} {
				u.refresh()
				u.key(tcell.NewEventKey(tcell.KeyBacktab, 0, 0))
				u.key(tcell.NewEventKey(tcell.KeyRune, 't', 0))
			}
			enter := tcell.NewEventKey(tcell.KeyEnter, 0, 0)
			right.input.SetText(tc.newValue)
			right.key(enter)
			left.key(enter)
			b, err := s.Read()
			if err != nil || b.Schedules[0].TimingValue() != tc.newValue {
				t.Fatalf("競合後の保存内容: %+v, %v", b, err)
			}
			if left.activePage() != "input" || left.input.GetText() != tc.newValue || !strings.Contains(left.notice.GetText(false), "別のペイン") {
				t.Fatalf("競合後の入力欄 = %q, 通知 = %q", left.input.GetText(), left.notice.GetText(false))
			}
			// 実行履歴だけが更新されても、最新の設定を見てやり直した編集は保存できる。
			if err := s.Edit(func(b *board.State) error {
				r := &b.Schedules[0]
				if r.Interval != nil {
					r.Interval.NextRun = r.Interval.NextRun.Add(time.Hour)
				} else {
					r.LastDate = time.Now().Format("2006-01-02")
				}
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			left.input.SetText(tc.nextValue)
			left.key(enter)
			b, err = s.Read()
			if err != nil || b.Schedules[0].TimingValue() != tc.nextValue || left.activePage() != "automation" {
				t.Fatalf("競合解消後の保存: %+v, %v", b, err)
			}
		})
	}
}
