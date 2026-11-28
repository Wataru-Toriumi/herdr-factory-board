package store

import (
	"testing"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
)

func TestChangeCardProjectRejectsStaleSelection(t *testing.T) {
	s := Store{Dir: t.TempDir()}
	var seen board.Card
	if err := s.Edit(func(b *board.State) error {
		seen = *b.NewCard("確認", "/api", "")
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.ChangeCardProject(seen, "/web"); err != nil {
		t.Fatal(err)
	}
	if err := s.ChangeCardProject(seen, "/docs"); err == nil {
		t.Fatal("変更前のカードを使って作業先を上書きできました")
	}
	b, err := s.Read()
	if err != nil {
		t.Fatal(err)
	}
	got, err := b.FindCard(seen.ID)
	if err != nil {
		t.Fatal(err)
	}
	want := seen
	want.Dir = "/web"
	if !got.Equal(want) {
		t.Fatalf("保存されたカード = %+v, 期待値 = %+v", *got, want)
	}
}
