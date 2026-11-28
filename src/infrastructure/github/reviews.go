// Package github は認証済みの GitHub CLI でレビュー依頼を取得する。
package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type PullRequest struct {
	Title, URL string
	Number     int
	Repository struct{ NameWithOwner string }
}

func RequestedReviews(ctx context.Context) ([]PullRequest, error) {
	out, err := run(ctx, "search", "prs", "--review-requested=@me", "--state=open", "--sort=updated", "--limit=1000", "--json", "repository,number,title,url")
	if err != nil {
		return nil, err
	}
	var prs []PullRequest
	if err := json.Unmarshal(out, &prs); err != nil {
		return nil, fmt.Errorf("レビュー依頼の読み込みに失敗しました: %w", err)
	}
	return prs, nil
}

func OpenPullRequest(ctx context.Context, url string) error {
	_, err := run(ctx, "pr", "view", "--web", "--", url)
	return err
}

func run(ctx context.Context, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "gh", args...).Output()
	if err == nil {
		return out, nil
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if errors.Is(err, exec.ErrNotFound) {
		return nil, errors.New("GitHub CLI（gh）をインストールしてください")
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		if exit.ExitCode() == 4 {
			return nil, errors.New("gh auth login で GitHub にログインしてください")
		}
		if msg := strings.Join(strings.Fields(string(exit.Stderr)), " "); msg != "" {
			return nil, errors.New(msg)
		}
	}
	return nil, err
}
