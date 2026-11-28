package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/store"
	"github.com/gdamore/tcell/v2"
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
		u.app.QueueEvent(tcell.NewEventKey(tcell.KeyRune, 'q', 0))
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
