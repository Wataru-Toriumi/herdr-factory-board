package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/application/scheduler"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/application/worker"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/store"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/tui"
)

func run() error {
	s, err := store.ForWorkspace()
	if err != nil {
		return err
	}
	herdr := os.Getenv("HERDR_BIN_PATH")
	if herdr == "" {
		herdr = "herdr"
	}
	if len(os.Args) == 1 {
		return tui.Run(s, herdr)
	}
	switch os.Args[1] {
	case "chat-reply":
		return worker.RunChatReply(s, os.Args[2:])
	case "worker":
		return worker.Run(s)
	case "scheduler":
		return scheduler.Run(s, herdr)
	default:
		return errors.New("不明な起動モードです")
	}
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
