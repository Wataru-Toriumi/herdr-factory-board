package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/application/scheduler"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/herdr"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/store"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/tui/agentlist"
)

type boardSnapshot struct {
	readAt                time.Time
	state                 board.State
	active                bool
	schedulerErr, readErr error
}

func readBoard(s store.Store) boardSnapshot {
	b := boardSnapshot{readAt: time.Now()}
	b.state, b.readErr = s.Read()
	if b.readErr == nil {
		b.active, b.schedulerErr = scheduler.Running(s)
	}
	return b
}

type sessionSnapshot struct {
	panes  []herdr.Pane
	spaces []herdr.Workspace
	err    error
}

func readSession(ctx context.Context, bin string) sessionSnapshot {
	panes, err := herdr.ListPanesContext(ctx, bin)
	if err != nil {
		return sessionSnapshot{err: fmt.Errorf("ペイン一覧: %w", err)}
	}
	spaces, err := herdr.ListWorkspacesContext(ctx, bin)
	if err != nil {
		return sessionSnapshot{panes: panes, err: fmt.Errorf("ワークスペース一覧: %w", err)}
	}
	return sessionSnapshot{panes: panes, spaces: spaces}
}

func (u *boardUI) poll(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		session := readSession(ctx, u.herdr)
		if ctx.Err() != nil {
			return
		}
		b := readBoard(u.store)
		if ctx.Err() != nil {
			return
		}
		u.app.QueueUpdateDraw(func() { u.applyRefresh(b, session) })
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (u *boardUI) applyRefresh(b boardSnapshot, session sessionSnapshot) {
	if session.panes != nil {
		u.panes = session.panes
	}
	if session.spaces != nil {
		u.spaces = session.spaces
	}
	u.renderBoard(b)

	// バックグラウンドの通知で、直前の操作結果を上書きしない。
	message := ""
	if session.err != nil {
		message = session.err.Error()
	}
	if notice := u.notice.GetText(false); notice == "" || notice == u.sessionError {
		u.notice.SetText(message)
	}
	u.sessionError = message
}

// 操作直後は保存済みボードを読み直し、Herdr の一覧には直近の取得結果を使う。
func (u *boardUI) refresh() {
	u.renderBoard(readBoard(u.store))
}

func (u *boardUI) renderBoard(snapshot boardSnapshot) {
	// 取得中にユーザーが編集した場合は、操作後に読み直した状態を優先する。
	if !snapshot.readAt.Before(u.board.readAt) {
		if snapshot.readErr != nil {
			u.notice.SetText(snapshot.readErr.Error())
			return
		}
		u.board = snapshot
	}
	b := u.board.state
	u.projects = b.ProjectDirs()
	u.schedules.render(b.Schedules, u.workDir, u.board.active, u.board.schedulerErr)
	if u.activePage() == "detail" {
		u.refreshDetail(b)
	}
	u.renderTasks(b.Cards)
	if u.board.schedulerErr != nil {
		u.notice.SetText(u.board.schedulerErr.Error())
	}
	u.showWorkDir()
	u.agents.render(agentlist.Build(u.panes, u.spaces, b.Cards))
	u.updateHeader()
}
