package worker

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/store"
)

// 相談の実行中だけ使う識別子。カードの作業フェーズには保存しない。
const chatPhase = "chat"

func chatPrompt(c board.Card) string {
	return fmt.Sprintf("チケットについての相談です。作業先を読み取り専用で調査し、最後の質問に日本語で回答してください。実装やファイル変更は行わないでください。\n\n元の依頼\n%s\n\nカードの状態\n%s\n\n設計案\n%s\n\n実行結果\n%s\n\nこれまでの会話\n%s\n\n今回の質問\n%s",
		c.Prompt, c.Status, c.Design, c.Result, c.Discussion(), c.Chat[len(c.Chat)-1].Question)
}

// 質問を先に保存し、同じチケットの排他ロックを応答プロセスに引き継ぐ。
func SendChat(s store.Store, seen board.Card, question string) error {
	question = strings.TrimSpace(question)
	if question == "" {
		return errors.New("メッセージを入力してください")
	}
	lock, err := s.Lock(fmt.Sprintf("chat-%d.lock", seen.ID), true)
	if errors.Is(err, syscall.EWOULDBLOCK) {
		return errors.New("担当が回答中です。回答が届いてから送信してください")
	}
	if err != nil {
		return err
	}
	defer lock.Close()
	var turn int
	err = s.EditCard(seen.ID, func(c *board.Card) error {
		if !c.CanChat() || !c.Equal(seen) {
			return errors.New("最新の手動チケットを確認してから送信してください")
		}
		if n := len(c.Chat); n > 0 && c.Chat[n-1].Pending() {
			// ロックを取得できたので、前回の応答プロセスは既に終了している。
			c.Chat[n-1].Error = "回答が中断されました"
		}
		c.Chat = append(c.Chat, board.ChatTurn{Question: question, Agent: c.Agent})
		turn = len(c.Chat)
		return nil
	})
	if err != nil {
		return err
	}
	exe, err := os.Executable()
	if err == nil {
		cmd := exec.Command(exe, "chat-reply", strconv.Itoa(seen.ID), strconv.Itoa(turn))
		cmd.Dir = seen.Dir
		cmd.ExtraFiles = []*os.File{lock}
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		if err = cmd.Start(); err == nil {
			go func() {
				if err := cmd.Wait(); err != nil {
					_ = finishChat(s, seen.ID, turn, board.Failed, err.Error())
				}
			}()
			return nil
		}
	}
	return errors.Join(err, finishChat(s, seen.ID, turn, board.Failed, err.Error()))
}

func finishChat(s store.Store, id, turn int, status, text string) error {
	return s.EditCard(id, func(c *board.Card) error {
		if turn != len(c.Chat) || !c.Chat[turn-1].Pending() {
			return nil
		}
		t := &c.Chat[turn-1]
		if status == board.Done {
			t.Reply = text
		} else {
			t.Error = text
			if t.Error == "" {
				t.Error = "回答が中断されました"
			}
		}
		return nil
	})
}

// ボードを閉じた後も応答を受け取り、質問と同じチケットに保存する。
func RunChatReply(s store.Store, args []string) error {
	if len(args) != 2 {
		return errors.New("チケットの会話欄から起動してください")
	}
	id, err := strconv.Atoi(args[0])
	if err != nil {
		return err
	}
	turn, err := strconv.Atoi(args[1])
	if err != nil || turn < 1 {
		return errors.New("会話番号が不正です")
	}
	lock := os.NewFile(3, "chat.lock")
	if lock == nil {
		return errors.New("相談のロックがありません")
	}
	defer lock.Close()
	b, err := s.Read()
	if err != nil {
		return err
	}
	c, err := b.FindCard(id)
	if err != nil {
		return err
	}
	if !c.CanChat() || turn != len(c.Chat) || !c.Chat[turn-1].Pending() {
		return errors.New("この相談は終了しています")
	}
	file, err := os.CreateTemp(s.Dir, ".chat-result-*")
	if err != nil {
		return finishChat(s, id, turn, board.Failed, err.Error())
	}
	defer os.Remove(file.Name())
	if err := file.Close(); err != nil {
		return finishChat(s, id, turn, board.Failed, err.Error())
	}
	task := *c
	// 別のペインで作業先を変更しても、送信時のプロジェクトで回答する。
	task.Dir, err = os.Getwd()
	if err != nil {
		return finishChat(s, id, turn, board.Failed, err.Error())
	}
	task.Agent, task.Phase = task.Chat[turn-1].Agent, chatPhase
	status, answer := runAgent(context.Background(), task, lock, file.Name())
	return finishChat(s, id, turn, status, answer)
}
