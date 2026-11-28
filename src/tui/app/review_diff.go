package app

import (
	"context"
	"fmt"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/github"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/tui/reviewlist"
	"github.com/rivo/tview"
)

func (u *boardUI) openReviewDiff(pr github.PullRequest) {
	ctx, cancel := context.WithCancel(context.Background())
	u.cancelReviewDiff = cancel
	view := tview.NewTextView().SetWrap(false).SetDynamicColors(true)
	view.SetBorder(true).SetTitle(" PR の差分 ")
	view.SetText("GitHub から差分を取得中…")
	u.pages.AddAndSwitchToPage("review-diff", view, true)
	u.app.SetFocus(view)
	u.context.SetText(fmt.Sprintf("%s#%d  %s", pr.Repository.NameWithOwner, pr.Number, pr.Title))
	u.hint.SetText("↑↓←→ スクロール  PgUp/PgDn ページ移動  Esc 一覧へ")

	go func() {
		diff, err := github.PullRequestDiff(ctx, pr.URL)
		if ctx.Err() != nil {
			return
		}
		var body string
		if err != nil {
			body = "差分を取得できませんでした。\n" + tview.Escape(err.Error())
		} else {
			body = reviewlist.DiffText(diff)
		}
		u.app.QueueUpdateDraw(func() {
			if ctx.Err() == nil {
				view.SetText(body).ScrollToBeginning()
			}
		})
	}()
}
