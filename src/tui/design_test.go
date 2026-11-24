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
	"github.com/Wataru-Toriumi/herdr-factory-board/src/tui/taskboard"
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
	if c := u.detail.Card(); u.tasks.Lane() != taskboard.Design || c.Status != board.Starting || c.Phase != board.DesignPhase {
		t.Fatalf("設計開始: %+v", c)
	}
	if prompt := run(1, "design", "設計案A"); !strings.Contains(prompt, "検索を追加") {
		t.Fatal(prompt)
	}
	u.refresh()
	first := *u.detail.Card()
	if u.tasks.Lane() != taskboard.DesignComplete || !first.CanReviewDesign() || first.Design != "設計案A" || first.Attempt != 1 || !strings.Contains(u.hint.GetText(false), "承認") {
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
	if u.activePage() != "detail" || u.detail.Card().Status != board.AwaitingDesignReview {
		t.Fatal("キャンセル後のレビュー状態が不正です")
	}
	key('r')
	u.input.SetText("索引を使う")
	press(tcell.KeyEnter)
	if u.tasks.Lane() != taskboard.Design || u.detail.Card().Status != board.Starting {
		t.Fatalf("再設計開始: %+v", u.detail.Card())
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
	for i := taskboard.Todo; i < taskboard.DesignComplete; i++ {
		press(tcell.KeyRight)
	}
	press(tcell.KeyEnter)
	if c := u.detail.Card(); c == nil || c.Design != "設計案B" || c.Status != board.AwaitingDesignReview || !strings.Contains(u.detail.GetText(false), "設計完了") {
		t.Fatalf("再設計結果: %+v", c)
	}
	key('a')
	if u.detail.Card().Status != board.Starting || taskboard.LaneFor(*u.detail.Card()) != taskboard.Work {
		t.Fatalf("実装開始時の列: %+v", u.detail.Card())
	}
	key('a')
	prompt = run(3, "implementation", "実装とテストが完了")
	if !strings.Contains(prompt, "検索を追加") || !strings.Contains(prompt, "設計案B") || strings.Contains(prompt, "設計案A") {
		t.Fatal(prompt)
	}
	u.refresh()
	if c := u.detail.Card(); c.Status != board.AwaitingResultReview || taskboard.LaneFor(*c) != taskboard.Review || !strings.Contains(u.hint.GetText(false), "承認して完了") {
		t.Fatalf("実装結果のレビュー: %+v", c)
	}
	review := *u.detail.Card()
	if err := worker.Start(s, "true", review.ID); err == nil {
		t.Fatal("レビュー中に実装が再起動されました")
	}
	u = newBoardUI(s, "true")
	u.refresh()
	for i := 0; i < taskboard.Review; i++ {
		press(tcell.KeyRight)
	}
	press(tcell.KeyEnter)
	if u.activePage() != "detail" || !strings.Contains(u.detail.GetText(false), "実装とテストが完了") {
		t.Fatal("レビュー中のカードで結果を確認できません")
	}
	key('a')
	if u.tasks.Lane() != taskboard.Done {
		t.Fatal("承認後に完了列へ移動しませんでした")
	}
	if err := worker.ApproveResult(s, review); err == nil {
		t.Fatal("同じレビューを二度承認できました")
	}
	if c := u.detail.Card(); c.Status != board.Done || c.Attempt != 3 || c.Design != "設計案B" || c.Result != "実装とテストが完了" || c.Phase != board.ImplementationPhase {
		t.Fatalf("実装結果: %+v", c)
	}

	key('u')
	if c := u.detail.Card(); u.tasks.Lane() != taskboard.Review || c.Result != review.Result || c.Attempt != review.Attempt {
		t.Fatalf("承認取り消し後: %+v", c)
	}
	if err := worker.ApproveResult(s, review); err == nil {
		t.Fatal("取り消し前の画面から再承認できました")
	}
	key('r')
	press(tcell.KeyEnter)
	if u.activePage() != "input" {
		t.Fatal("空の実装修正依頼が送信されました")
	}
	press(tcell.KeyEscape)
	if u.activePage() != "detail" || !u.detail.Card().CanReviewResult() {
		t.Fatal("修正依頼の取り消し後にレビューを続けられません")
	}
	key('r')
	u.input.SetText("結果が空の場合も検証して")
	press(tcell.KeyEnter)
	if u.tasks.Lane() != taskboard.Work || u.detail.Card().Status != board.Starting {
		t.Fatalf("実装差し戻し後: %+v", u.detail.Card())
	}
	prompt = run(4, "implementation", "空の検索結果も検証済み")
	for _, part := range []string{"検索を追加", "設計案B", review.Result, "結果が空の場合も検証して"} {
		if !strings.Contains(prompt, part) {
			t.Fatalf("差し戻し後のプロンプトに %q がありません: %s", part, prompt)
		}
	}
	u.refresh()
	if u.tasks.Lane() != taskboard.Review || u.detail.Card().Result != "空の検索結果も検証済み" {
		t.Fatalf("修正後のレビュー: %+v", u.detail.Card())
	}
	key('a')
	if u.tasks.Lane() != taskboard.Done {
		t.Fatal("修正後の結果を承認できません")
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
	u.openCardDetail(board.Card{ID: 1, Status: board.AwaitingDesignReview, Phase: board.DesignPhase})
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

func TestReopenedCardCanStartFresh(t *testing.T) {
	s := store.Store{Dir: t.TempDir()}
	err := s.Edit(func(b *board.State) error {
		c := b.NewCard("手動", s.Dir, "")
		c.Status, c.Phase, c.Attempt = board.Canceled, board.ImplementationPhase, 3
		c.Design, c.Feedback, c.ImplementationFeedback = "以前の設計", "設計の指摘", "実装の指摘"
		c.PreviousResult, c.Result = "以前の結果", "中断時の結果"
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	u := newBoardUI(s, "true")
	u.execute("result 1")
	u.key(tcell.NewEventKey(tcell.KeyRune, 'u', 0))
	if c := u.detail.Card(); c.Phase != "" || c.Result != "中断時の結果" || c.Design != "以前の設計" {
		t.Fatalf("未着手に戻した直後の記録: %+v", c)
	}
	u.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	u.key(tcell.NewEventKey(tcell.KeyEnter, 0, 0))
	b, err := s.Read()
	if err != nil {
		t.Fatal(err)
	}
	c := b.Cards[0]
	if c.Status != board.Starting || c.Attempt != 4 || c.Phase != "" || c.Design != "" || c.Feedback != "" || c.ImplementationFeedback != "" || c.PreviousResult != "" || c.Result != "" {
		t.Fatalf("直接実行の開始時に以前の状態が残りました: %+v", c)
	}
}

func TestFeedbackEditorKeepsReviewedDesignDuringRefresh(t *testing.T) {
	s := store.Store{Dir: t.TempDir()}
	if err := s.Edit(func(b *board.State) error {
		c := b.NewCard("設計を確認", s.Dir, "")
		c.Phase, c.Status, c.Design = board.DesignPhase, board.AwaitingDesignReview, "最初の設計案"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	u := newBoardUI(s, "true")
	u.execute("result 1")
	u.key(tcell.NewEventKey(tcell.KeyRune, 'r', 0))
	u.input.SetText("条件を追記")
	if err := s.EditCard(1, func(c *board.Card) error {
		c.Design = "別ペインの設計案"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	u.refresh()
	u.key(tcell.NewEventKey(tcell.KeyEnter, 0, 0))
	if u.activePage() != "detail" || u.app.GetFocus() != u.detail || !strings.Contains(u.notice.GetText(false), "カードが更新") {
		t.Fatalf("古い設計案への送信後: %s, %s", u.activePage(), u.notice.GetText(false))
	}
	if c := u.detail.Card(); c.Design != "別ペインの設計案" || c.Status != board.AwaitingDesignReview || c.Feedback != "" {
		t.Fatalf("最新のカード: %+v", c)
	}
}
