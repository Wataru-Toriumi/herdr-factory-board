package tui

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/application/worker"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/store"
	"github.com/gdamore/tcell/v2"
)

func TestDesignReviewFromTaskDetail(t *testing.T) {
	s := store.Store{Dir: t.TempDir()}
	if err := s.Edit(func(b *board.State) error {
		b.NewCard("検索を追加", s.Dir, "")
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	script := `#!/bin/sh
set -eu
test "$1" = exec
test "$2" = --output-last-message
result=$3
shift 3
if test "$EXPECT_PHASE" = design; then
	test "$1" = --sandbox
	test "$2" = read-only
	shift 2
fi
test "$#" = 1 && test "$1" = -
cat > "$CAPTURE"
printf '%s' "$ANSWER" > "$result"
`
	if err := os.WriteFile(filepath.Join(bin, "codex"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+":"+os.Getenv("PATH"))
	capture := filepath.Join(bin, "prompt")
	t.Setenv("CAPTURE", capture)
	t.Setenv("FACTORY_CARD", "1")
	t.Setenv("HERDR_PANE_ID", "worker")
	run := func(attempt int, phase, answer string) string {
		t.Helper()
		t.Setenv("EXPECT_PHASE", phase)
		t.Setenv("ANSWER", answer)
		t.Setenv("FACTORY_ATTEMPT", strconv.Itoa(attempt))
		if err := worker.Run(s); err != nil {
			t.Fatal(err)
		}
		text, err := os.ReadFile(capture)
		if err != nil {
			t.Fatal(err)
		}
		return string(text)
	}
	u := newBoardUI(s, "true")
	u.refresh()
	key := func(r rune) { u.key(tcell.NewEventKey(tcell.KeyRune, r, 0)) }
	press := func(k tcell.Key) { u.key(tcell.NewEventKey(k, 0, 0)) }
	key('v')
	key('d')
	if c := u.detailCard; u.lane != designLane || c.Status != board.Starting || c.Phase != board.DesignPhase {
		t.Fatalf("設計開始: %+v", c)
	}
	if prompt := run(1, "design", "設計案A"); !strings.Contains(prompt, "検索を追加") {
		t.Fatal(prompt)
	}
	u.refresh()
	first := *u.detailCard
	if u.lane != designCompleteLane || !first.CanReviewDesign() || first.Design != "設計案A" || first.Attempt != 1 || !strings.Contains(u.hint.GetText(false), "承認") {
		t.Fatalf("設計レビュー: %+v", first)
	}
	if err := worker.Start(s, "true", 1); err == nil {
		t.Fatal("レビュー中に通常実行できました")
	}
	key('r')
	press(tcell.KeyEnter)
	if u.activePage() != "input" {
		t.Fatal("空のコメントが受理されました")
	}
	press(tcell.KeyEscape)
	if u.activePage() != "detail" || u.detailCard.Status != board.AwaitingDesignReview {
		t.Fatal("キャンセル後のレビュー状態が不正です")
	}
	key('r')
	u.input.SetText("索引を使う")
	press(tcell.KeyEnter)
	if u.lane != designLane || u.detailCard.Status != board.Starting {
		t.Fatalf("再設計開始: %+v", u.detailCard)
	}
	prompt := run(2, "design", "設計案B")
	for _, part := range []string{"検索を追加", "設計案A", "索引を使う"} {
		if !strings.Contains(prompt, part) {
			t.Fatalf("再設計の入力: %s", prompt)
		}
	}
	if err := worker.StartPhase(s, "true", first, board.ImplementationPhase, ""); err == nil {
		t.Fatal("古い設計案を承認できました")
	}

	// ボードを開き直しても、同じカードの最新版を承認できる。
	u = newBoardUI(s, "true")
	u.refresh()
	for i := todoLane; i < designCompleteLane; i++ {
		press(tcell.KeyRight)
	}
	press(tcell.KeyEnter)
	if c := u.detailCard; c == nil || c.Design != "設計案B" || c.Status != board.AwaitingDesignReview || !strings.Contains(u.detail.GetText(false), "設計完了") {
		t.Fatalf("再設計結果: %+v", c)
	}
	key('a')
	if u.detailCard.Status != board.Starting || cardLane(*u.detailCard) != workLane {
		t.Fatalf("実装開始時の列: %+v", u.detailCard)
	}
	key('a')
	prompt = run(3, "implementation", "実装とテストが完了")
	if !strings.Contains(prompt, "検索を追加") || !strings.Contains(prompt, "設計案B") || strings.Contains(prompt, "設計案A") {
		t.Fatal(prompt)
	}
	u.refresh()
	if c := u.detailCard; c.Status != board.AwaitingResultReview || cardLane(*c) != reviewLane || !strings.Contains(u.hint.GetText(false), "承認して完了") {
		t.Fatalf("実装結果のレビュー: %+v", c)
	}
	review := *u.detailCard
	if err := worker.Start(s, "true", review.ID); err == nil {
		t.Fatal("レビュー中に実装が再起動されました")
	}
	u = newBoardUI(s, "true")
	u.refresh()
	for i := 0; i < reviewLane; i++ {
		press(tcell.KeyRight)
	}
	press(tcell.KeyEnter)
	if u.activePage() != "detail" || !strings.Contains(u.detail.GetText(false), "実装とテストが完了") {
		t.Fatal("レビュー中のカードで結果を確認できません")
	}
	key('a')
	if u.lane != doneLane {
		t.Fatal("承認後に完了列へ移動しませんでした")
	}
	if err := worker.ApproveResult(s, review); err == nil {
		t.Fatal("同じレビューを二度承認できました")
	}
	if c := u.detailCard; c.Status != board.Done || c.Attempt != 3 || c.Design != "設計案B" || c.Result != "実装とテストが完了" || c.Phase != board.ImplementationPhase {
		t.Fatalf("実装結果: %+v", c)
	}
}

func TestDesignFailureRequiresNewDesignBeforeApproval(t *testing.T) {
	s := store.Store{Dir: t.TempDir()}
	if err := s.Edit(func(b *board.State) error {
		b.NewCard("手動", s.Dir, "")
		b.NewCard("定期", s.Dir, "定期 #5")
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	b, err := s.Read()
	if err != nil {
		t.Fatal(err)
	}
	if err := worker.StartPhase(s, "true", b.Cards[1], board.DesignPhase, ""); err == nil {
		t.Fatal("定期カードが設計モードに入りました")
	}
	if err := worker.StartPhase(s, "true", b.Cards[0], board.DesignPhase, ""); err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "codex"), []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+":"+os.Getenv("PATH"))
	t.Setenv("FACTORY_CARD", "1")
	t.Setenv("FACTORY_ATTEMPT", "1")
	if err := worker.Run(s); err != nil {
		t.Fatal(err)
	}
	b, err = s.Read()
	if err != nil {
		t.Fatal(err)
	}
	c := b.Cards[0]
	if c.Status != board.Failed || c.Design != "" || !c.CanStartDesign() {
		t.Fatalf("設計案が空の実行: %+v", c)
	}
	if err := worker.StartPhase(s, "true", c, board.ImplementationPhase, ""); err == nil {
		t.Fatal("失敗した設計を承認できました")
	}
	if err := worker.Start(s, "true", c.ID); err != nil {
		t.Fatal(err)
	}
	b, err = s.Read()
	if err != nil || b.Cards[0].Phase != board.DesignPhase {
		t.Fatal("設計として再試行されませんでした")
	}
}

func TestIncompleteDesignHasNoReviewShortcuts(t *testing.T) {
	u := newBoardUI(store.Store{Dir: t.TempDir()}, "true")
	u.read("カードの詳細", "", 0)
	u.renderCardDetail(board.Card{ID: 1, Status: board.AwaitingDesignReview, Phase: board.DesignPhase})
	if hint := u.hint.GetText(false); strings.Contains(hint, "承認して実装") || strings.Contains(hint, "再設計") {
		t.Fatalf("設計案が空なのにレビュー操作が表示されました: %s", hint)
	}
	for _, key := range []rune{'a', 'r'} {
		event := tcell.NewEventKey(tcell.KeyRune, key, 0)
		if u.key(event) != event {
			t.Fatalf("設計案が空なのに %q を受け付けました", key)
		}
	}
}
