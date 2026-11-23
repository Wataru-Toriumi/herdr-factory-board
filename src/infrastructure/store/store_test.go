package store

import (
	"testing"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
)

func TestConcurrentEditsPreserveCards(t *testing.T) {
	s := Store{Dir: t.TempDir()}
	results := make(chan error, 10)
	for i := 0; i < cap(results); i++ {
		go func() {
			results <- s.Edit(func(b *board.State) error { b.NewCard("task", s.Dir, ""); return nil })
		}()
	}
	for i := 0; i < cap(results); i++ {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	b, err := s.Read()
	if err != nil || len(b.Cards) != 10 || b.NextID != 10 {
		t.Fatalf("lost edit: %+v, %v", b, err)
	}
}
