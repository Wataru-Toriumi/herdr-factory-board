// Package reviewlist は GitHub のレビュー依頼をリポジトリ横断で表示する。
package reviewlist

import (
	"fmt"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/github"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/tui/widgets"
	"github.com/gdamore/tcell/v2"
)

const ListHint = "↑↓ 選択  Enter GitHubで開く  r 更新  Tab 切替  q 閉じる"

type List struct {
	*widgets.RowList[github.PullRequest]
	refresh func()
	open    func(string)
}

func New(refresh func(), open func(string)) *List {
	l := &List{RowList: widgets.NewRowList[github.PullRequest](), refresh: refresh, open: open}
	l.SetTitle(" GitHub のレビュー依頼 ")
	return l
}

func (l *List) Render(prs []github.PullRequest) {
	l.SetRows(prs, func(a, b github.PullRequest) bool { return a.URL == b.URL }, func(pr github.PullRequest) (string, string) {
		return pr.Title, fmt.Sprintf("%s#%d", pr.Repository.NameWithOwner, pr.Number)
	})
	l.SetTitle(fmt.Sprintf(" GitHub のレビュー依頼 %d件 ", len(prs)))
}

func (l *List) HandleKey(event *tcell.EventKey) bool {
	switch {
	case event.Key() == tcell.KeyEnter:
		if pr, ok := l.Selected(); ok {
			l.open(pr.URL)
		}
	case event.Key() == tcell.KeyRune && event.Rune() == 'r':
		l.refresh()
	default:
		return false
	}
	return true
}
