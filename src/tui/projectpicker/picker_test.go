package projectpicker

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestBrowseAndRetryRegistration(t *testing.T) {
	parent := t.TempDir()
	child := filepath.Join(parent, "api")
	if err := os.Mkdir(child, 0700); err != nil {
		t.Fatal(err)
	}

	var list *tview.List
	var shown, chosen string
	registrationErr := errors.New("登録失敗")
	var reported error
	picker := Picker{
		Show: func(view *tview.List, path, hint string) { list, shown = view, path },
		Register: func(path string) error {
			if path != child {
				t.Fatalf("登録先 = %q", path)
			}
			return registrationErr
		},
		ReportError: func(err error) { reported = err },
	}
	picker.Open(nil, parent, false, func(path string) { chosen = path })
	enter := func(label string) {
		t.Helper()
		for i := 0; i < list.GetItemCount(); i++ {
			if text, _ := list.GetItemText(i); text == label {
				list.SetCurrentItem(i)
				list.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), func(tview.Primitive) {})
				return
			}
		}
		t.Fatalf("一覧に %q がありません", label)
	}
	enter("api/")
	if shown != child {
		t.Fatalf("表示中のフォルダ = %q", shown)
	}
	enter("このフォルダを作業先にする")
	if reported != registrationErr || chosen != "" {
		t.Fatalf("登録失敗時の通知 = %v, 選択 = %q", reported, chosen)
	}
	registrationErr = nil
	enter("このフォルダを作業先にする")
	if chosen != child {
		t.Fatalf("登録後の選択 = %q", chosen)
	}
}
