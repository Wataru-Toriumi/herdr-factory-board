// Package dictation は音声入力ヘルパーの録音と文字起こしを管理する。
package dictation

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type Update struct {
	Ready bool
	Text  string
	Err   error
}

type Session struct {
	Updates <-chan Update
	input   io.WriteCloser
	cancel  context.CancelFunc
}

func HelperPath() string {
	if runtime.GOOS != "darwin" {
		return ""
	}
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Join(filepath.Dir(exe), "factory-board-dictation")
}

func Start(helper string) (*Session, error) {
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, helper)
	var diagnostic bytes.Buffer
	cmd.Stderr = &diagnostic
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	input, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		_ = stdout.Close()
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("音声入力を起動できません。プラグインを再ビルドしてください: %w", err)
	}

	updates := make(chan Update, 2)
	go func() {
		defer close(updates)
		defer cancel()
		reader := bufio.NewReader(stdout)
		header, _ := reader.ReadString('\n')
		if header == "ready\n" {
			updates <- Update{Ready: true}
		}
		body, readErr := io.ReadAll(reader)
		err := cmd.Wait()
		text := strings.TrimSpace(string(body))
		switch {
		case ctx.Err() != nil:
			return
		case err != nil:
			if message := strings.TrimSpace(diagnostic.String()); message != "" {
				err = errors.New(message)
			}
		case readErr != nil:
			err = readErr
		case header != "ready\n" || text == "":
			err = errors.New("音声を認識できませんでした")
		}
		updates <- Update{Text: text, Err: err}
	}()
	return &Session{Updates: updates, input: input, cancel: cancel}, nil
}

func (s *Session) Stop() error {
	_, err := io.WriteString(s.input, "stop\n")
	return err
}

func (s *Session) Cancel() { s.cancel() }
