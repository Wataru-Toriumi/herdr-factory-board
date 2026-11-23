package tui

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/application/scheduler"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/application/worker"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/schedule"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/store"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestProjectSelectionAcrossPanes(t *testing.T) {
	s := store.Store{Dir: t.TempDir()}
	api := filepath.Join(s.Dir, "api")
	web := filepath.Join(s.Dir, "web")
	for _, dir := range []string{api, web} {
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Edit(func(b *board.State) error {
		b.Dir = api
		b.NewCard("APIタスク", api, "")
		b.NewCard("Webタスク", web, "")
		b.NextID++
		b.Schedules = []schedule.Rule{{ID: b.NextID, Time: "08:00", Dir: web, Prompt: "Web確認"}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	left, right := newBoardUI(s, "true"), newBoardUI(s, "true")
	left.refresh()
	right.refresh()
	key := func(u *boardUI, ch rune) { u.key(tcell.NewEventKey(tcell.KeyRune, ch, 0)) }
	press := func(u *boardUI, k tcell.Key) { u.key(tcell.NewEventKey(k, 0, 0)) }
	choose := func(u *boardUI, name string) {
		t.Helper()
		page, view := u.pages.GetFrontPage()
		list, ok := view.(*tview.List)
		if page != "directory" || !ok {
			t.Fatal("プロジェクトの選択画面が開いていません")
		}
		for i := 0; i < list.GetItemCount(); i++ {
			main, _ := list.GetItemText(i)
			if main == name {
				list.SetCurrentItem(i)
				list.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), func(p tview.Primitive) { u.app.SetFocus(p) })
				return
			}
		}
		t.Fatalf("選択肢に %q がありません", name)
	}

	if !slices.Equal(left.projects, []string{api, web}) || len(left.cards[todoLane]) != 2 {
		t.Fatal("旧版のボードからプロジェクトを復元できません")
	}
	_, label := left.lists[todoLane].GetItemText(1)
	if !strings.Contains(label, "web") {
		t.Fatalf("全体表示のカードにプロジェクト名がありません: %s", label)
	}

	key(left, 'p')
	choose(left, "api")
	key(left, 'n')
	choose(left, "web")
	left.input.SetText("add 左のタスク")
	right.execute("dir " + web)
	left.refresh()
	press(left, tcell.KeyEnter)
	if left.workDir != api || len(left.cards[todoLane]) != 1 || left.schedules.GetItemCount() != 0 {
		t.Fatal("APIのカードと予定に絞り込まれていません")
	}
	right.refresh()
	if len(right.cards[todoLane]) != 2 || right.schedules.GetItemCount() != 1 {
		t.Fatal("Webのカードと予定に絞り込まれていません")
	}

	right.lists[todoLane].SetCurrentItem(1)
	key(right, 'v')
	key(right, 'p')
	press(right, tcell.KeyEscape)
	if right.activePage() != "detail" || right.detailCard.Dir != web {
		t.Fatal("作業先の選択を取り消すと元のチケット詳細に戻れません")
	}
	key(right, 'p')
	choose(right, "api")
	if right.activePage() != "detail" || right.detailCard.Dir != api || right.workDir != web || len(right.cards[todoLane]) != 1 {
		t.Fatal("チケットの作業先変更が反映されません")
	}
	press(right, tcell.KeyEscape)

	key(left, 'p')
	choose(left, "すべて")
	press(left, tcell.KeyBacktab)
	key(left, 'i')
	choose(left, "api")
	left.input.SetText("every 15m API確認")
	press(left, tcell.KeyEnter)
	if left.workDir != "" || left.schedules.GetItemCount() != 2 {
		t.Fatal("全体表示から予定を登録できません")
	}
	right.execute("add 右のタスク")
	choose(right, "web")
	b, err := s.Read()
	if err != nil {
		t.Fatal(err)
	}
	if b.Cards[2].Dir != api || b.Cards[3].Dir != web || b.Schedules[1].Dir != api {
		t.Fatalf("ペインごとの登録先が不正です: %+v", b)
	}
	right.execute("result 2")
	key(right, 'p')
	if err := worker.Start(s, "true", 2); err != nil {
		t.Fatal(err)
	}
	choose(right, "api")
	if right.detailCard.Dir != web || right.detailCard.Status != board.Starting || !strings.Contains(right.notice.GetText(false), "更新") {
		t.Fatal("選択中に起動したカードの作業先を保護できません")
	}
	if err := scheduler.RunDue(s, "true", b.Schedules[1].Interval.NextRun); err != nil {
		t.Fatal(err)
	}
	b, err = s.Read()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range b.Cards {
		if c.ScheduleID == b.Schedules[1].ID && c.Dir == api && c.Status == board.Starting {
			return
		}
	}
	t.Fatal("APIの予定が登録先で起動しませんでした")
}

func TestRegisteredAndHistoricalProjects(t *testing.T) {
	s := store.Store{Dir: t.TempDir()}
	if err := s.RegisterProject(s.Dir); err != nil {
		t.Fatal(err)
	}
	if err := s.RegisterProject(s.Dir); err != nil {
		t.Fatal(err)
	}
	b, err := s.Read()
	if err != nil || !slices.Equal(b.Projects, []string{s.Dir}) {
		t.Fatalf("空のプロジェクトの保存: %+v, %v", b.Projects, err)
	}
	b.Cards = []board.Card{{Dir: "/backend/api"}}
	b.Schedules = []schedule.Rule{{Dir: "/frontend/api"}}
	names := projectNames(b.ProjectDirs())
	if names["/backend/api"] != "/backend/api" || names["/frontend/api"] != "/frontend/api" {
		t.Fatalf("同名のプロジェクトを区別できません: %v", names)
	}
}

func TestTypedScheduleProjectSelection(t *testing.T) {
	for _, tc := range []struct {
		name            string
		projects        []string
		filter, wantDir string
	}{
		{"全体表示に1件", []string{"/api"}, "", "/api"},
		{"個別表示に複数件", []string{"/api", "/web"}, "/web", "/web"},
		{"全体表示に複数件", []string{"/api", "/web"}, "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := store.Store{Dir: t.TempDir()}
			if err := s.Edit(func(b *board.State) error { b.Projects = tc.projects; return nil }); err != nil {
				t.Fatal(err)
			}
			u := newBoardUI(s, "true")
			u.workDir = tc.filter
			u.refresh()
			u.execute("  every   15m 差分を確認  ")
			if tc.wantDir == "" {
				if u.activePage() != "directory" {
					t.Fatal("登録先の選択が省略されました")
				}
				u.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
			}
			b, err := s.Read()
			if err != nil {
				t.Fatal(err)
			}
			if tc.wantDir == "" {
				if len(b.Schedules) != 0 {
					t.Fatal("選択を取り消した予定が登録されました")
				}
			} else if len(b.Schedules) != 1 || b.Schedules[0].Dir != tc.wantDir || b.Schedules[0].Prompt != "差分を確認" {
				t.Fatalf("登録後の予定: %+v", b.Schedules)
			}
		})
	}
}
