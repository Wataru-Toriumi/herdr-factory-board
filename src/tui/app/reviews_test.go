package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/store"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestGitHubReviewsAcrossRepositories(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))
	t.Setenv("REVIEW_FIXTURE", dir)
	script := `#!/bin/sh
case "$1 $2" in
  "search prs")
    test "$*" = "search prs --review-requested=@me --state=open --sort=updated --limit=1000 --json repository,number,title,url" || exit 2
    test ! -f "$REVIEW_FIXTURE/auth-error" || exit 4
    cat "$REVIEW_FIXTURE/prs"
    ;;
  "pr diff")
    test "$3" = --color=never && test "$4" = -- && test "$5" = https://github.com/team/web/pull/1 || exit 2
    cat "$REVIEW_FIXTURE/diff"
    ;;
  "pr view")
    test "$3" = --web && test "$4" = -- || exit 2
    echo "$5" > "$REVIEW_FIXTURE/opened"
    ;;
  *) exit 2 ;;
esac
`
	if err := os.WriteFile(filepath.Join(dir, "gh"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("prs", `[{"repository":{"nameWithOwner":"team/api"},"number":1,"title":"API","url":"https://github.com/team/api/pull/1"},{"repository":{"nameWithOwner":"team/web"},"number":1,"title":"UI","url":"https://github.com/team/web/pull/1"}]`)

	u := newBoardUI(store.Store{Dir: dir}, "true")
	u.app.SetScreen(tcell.NewSimulationScreen("UTF-8"))
	done := make(chan error, 1)
	go func() { done <- u.run() }()
	defer func() {
		u.app.QueueUpdateDraw(func() { u.app.Stop() })
		select {
		case err := <-done:
			if err != nil {
				t.Error(err)
			}
		case <-time.After(3 * time.Second):
			t.Error("ボードを終了できません")
		}
	}()
	u.app.QueueUpdateDraw(func() {
		u.workDir = "/unrelated"
		u.key(tcell.NewEventKey(tcell.KeyTab, 0, 0))
		u.key(tcell.NewEventKey(tcell.KeyTab, 0, 0))
	})
	waitFor := func(check func() bool) {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for {
			var ok bool
			var status string
			u.app.QueueUpdateDraw(func() {
				ok = !u.reviewBusy && check()
				status = u.reviewStatus
			})
			if ok {
				return
			}
			if time.Now().After(deadline) {
				t.Fatalf("レビュー一覧の更新を確認できません: %s", status)
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	waitFor(func() bool { return u.activePage() == "reviews" && u.reviews.GetItemCount() == 2 })
	u.app.QueueUpdateDraw(func() {
		u.reviews.SetCurrentItem(1)
		u.key(tcell.NewEventKey(tcell.KeyEnter, 0, 0))
	})
	waitFor(func() bool { return strings.Contains(u.reviewStatus, "開きました") })
	got, err := os.ReadFile(filepath.Join(dir, "opened"))
	if err != nil || strings.TrimSpace(string(got)) != "https://github.com/team/web/pull/1" {
		t.Fatalf("選択した PR の URL = %s, %v", got, err)
	}
	// 同じ番号でも選択中のリポジトリの差分を開き、戻ると選択を保持する。
	write("diff", "diff --git a/main.go b/main.go\n@@ -1 +1 @@\n-old\n+[red]更新\n")
	pressDiff := func() { u.app.QueueEvent(tcell.NewEventKey(tcell.KeyRune, 'd', 0)) }
	diffContains := func(want string) bool {
		page, pane := u.pages.GetFrontPage()
		return page == "review-diff" && strings.Contains(pane.(*tview.TextView).GetText(true), want)
	}
	back := func() {
		u.app.QueueEvent(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
		waitFor(func() bool {
			pr, ok := u.reviews.Selected()
			return u.activePage() == "reviews" && ok && pr.Repository.NameWithOwner == "team/web"
		})
	}
	pressDiff()
	waitFor(func() bool { return diffContains("+[red]更新") })
	back()
	write("diff", "")
	pressDiff()
	waitFor(func() bool { return diffContains("表示する差分はありません") })
	back()
	if err := os.Remove(filepath.Join(dir, "diff")); err != nil {
		t.Fatal(err)
	}
	pressDiff()
	waitFor(func() bool { return diffContains("差分を取得できませんでした") })
	back()
	refresh := func() { u.app.QueueUpdateDraw(func() { u.key(tcell.NewEventKey(tcell.KeyRune, 'r', 0)) }) }
	write("auth-error", "")
	refresh()
	waitFor(func() bool { return strings.Contains(u.reviewStatus, "gh auth login") && u.reviews.GetItemCount() == 2 })
	if err := os.Remove(filepath.Join(dir, "auth-error")); err != nil {
		t.Fatal(err)
	}
	write("prs", "[]")
	refresh()
	waitFor(func() bool {
		return strings.Contains(u.reviewStatus, "ありません") && u.reviews.GetItemCount() == 0
	})
}
