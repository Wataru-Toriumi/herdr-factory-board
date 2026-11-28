package app

import (
	"context"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/github"
)

// 空の URL は一覧の更新、それ以外は選択した PR を開く操作。
func (u *boardUI) requestReview(url string) {
	if u.reviewBusy {
		return
	}
	u.reviewBusy = true
	if url == "" {
		u.reviewStatus = "GitHub からレビュー依頼を取得中…"
	} else {
		u.reviewStatus = "ブラウザで PR を開いています…"
	}
	u.showWorkDir()
	u.reviewRequests <- url
}

func (u *boardUI) processReviews(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case url := <-u.reviewRequests:
			var prs []github.PullRequest
			var err error
			if url == "" {
				prs, err = github.RequestedReviews(ctx)
			} else {
				err = github.OpenPullRequest(ctx, url)
			}
			if ctx.Err() != nil {
				return
			}
			u.app.QueueUpdateDraw(func() {
				u.reviewBusy = false
				switch {
				case err != nil:
					u.reviewStatus = "GitHub: " + err.Error()
				case url != "":
					u.reviewStatus = "ブラウザで PR を開きました"
				default:
					u.reviews.Render(prs)
					u.reviewStatus = "自分へのレビュー依頼（全リポジトリ）"
					if len(prs) == 0 {
						u.reviewStatus = "自分へのレビュー依頼はありません"
					}
				}
				if u.activePage() == "reviews" {
					u.showWorkDir()
				}
			})
		}
	}
}
