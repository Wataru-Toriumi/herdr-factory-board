package main

import (
	"errors"
	"fmt"
	"os"
)

func run() error {
	s, err := workspaceStore()
	if err != nil {
		return err
	}
	herdr := os.Getenv("HERDR_BIN_PATH")
	if herdr == "" {
		herdr = "herdr"
	}
	if len(os.Args) == 1 {
		return runBoard(s, herdr)
	}
	switch os.Args[1] {
	case "worker":
		return runWorker(s)
	case "scheduler":
		return runScheduler(s, herdr)
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
