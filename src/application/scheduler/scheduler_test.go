package scheduler

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/application/worker"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/schedule"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/store"
)

func TestRunDueResumesPendingCardAndRecordsSimultaneousSchedules(t *testing.T) {
	s := store.Store{Dir: t.TempDir()}
	now := time.Date(2026, 11, 23, 9, 0, 0, 0, time.Local)
	err := s.Edit(func(b *board.State) error {
		b.Schedules = []schedule.Rule{
			{ID: 1, Dir: s.Dir, Time: "09:00"},
			{ID: 2, Dir: s.Dir, Time: "10:00"},
			{ID: 3, Dir: s.Dir, Interval: &schedule.Interval{Minutes: 15, NextRun: now}},
		}
		b.NextID = 3
		b.NewCard("保存済み", s.Dir, "定期 #1 2026-11-22 09:00").ScheduleID = 1
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := RunDue(s, "true", now.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	b, err := s.Read()
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Cards) != 1 || b.Cards[0].Status != board.Starting {
		t.Fatalf("予定時刻前の未起動カードの再開: %+v", b.Cards)
	}
	for i := 0; i < 2; i++ {
		if err := RunDue(s, "true", now); err != nil {
			t.Fatal(err)
		}
	}
	b, err = s.Read()
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Cards) != 3 || b.Cards[1].ScheduleID != 1 || b.Cards[2].ScheduleID != 3 {
		t.Fatalf("同時刻の予定から作成したカード: %+v", b.Cards)
	}
	for _, c := range b.Cards {
		if c.Status != board.Starting || c.Attempt != 1 {
			t.Fatalf("カードの起動状態: %+v", c)
		}
	}
}

func TestDailyOccurrenceAndRetry(t *testing.T) {
	s := store.Store{Dir: t.TempDir()}
	err := s.Edit(func(b *board.State) error {
		b.NextID = 1
		b.Schedules = []schedule.Rule{{ID: 1, Time: "09:00", Prompt: "Inspect project", Dir: s.Dir}}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	before := time.Date(2026, 11, 23, 8, 59, 0, 0, time.Local)
	if err := RunDue(s, "true", before); err != nil {
		t.Fatal(err)
	}
	b, err := s.Read()
	if err != nil || len(b.Cards) != 0 {
		t.Fatalf("early run: %+v, %v", b, err)
	}
	now := before.Add(time.Minute)
	if err := RunDue(s, "false", now); err == nil {
		t.Fatal("expected a launch failure")
	}
	if err := RunDue(s, "true", now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	b, err = s.Read()
	if err != nil || len(b.Cards) != 1 || b.Cards[0].Status != board.Failed {
		t.Fatalf("duplicate or missing failure: %+v, %v", b, err)
	}

	id := b.Cards[0].ID
	if err := worker.Start(s, "true", id); err != nil {
		t.Fatal(err)
	}
	if err := worker.Start(s, "true", id); err == nil {
		t.Fatal("duplicate start accepted")
	}
	t.Setenv("FACTORY_CARD", "2")
	t.Setenv("FACTORY_ATTEMPT", "1")
	if err := worker.Run(s); err == nil {
		t.Fatal("stale worker accepted")
	}

	bin := t.TempDir()
	err = os.WriteFile(filepath.Join(bin, "codex"), []byte("#!/bin/sh\ncat > \"$3\"\n"), 0700)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+":"+os.Getenv("PATH"))
	t.Setenv("FACTORY_ATTEMPT", "2")
	if err := worker.Run(s); err != nil {
		t.Fatal(err)
	}
	if err := RunDue(s, "true", now.AddDate(0, 0, 3)); err != nil {
		t.Fatal(err)
	}
	b, err = s.Read()
	if err != nil || len(b.Cards) != 2 || b.Cards[0].Status != board.Done || b.Cards[0].Result != "Inspect project" || b.Cards[1].ID == id {
		t.Fatalf("retry or catch-up failed: %+v, %v", b, err)
	}
}
