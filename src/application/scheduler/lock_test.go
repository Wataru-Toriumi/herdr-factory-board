package scheduler

import (
	"testing"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/store"
)

func TestSchedulerLockLifecycle(t *testing.T) {
	s := store.Store{Dir: t.TempDir()}
	checkRunning := func(want bool) {
		t.Helper()
		active, err := Running(s)
		if err != nil || active != want {
			t.Fatalf("稼働状態 = %v, %v; want %v", active, err, want)
		}
	}
	checkRunning(false)
	lock, err := tryAcquireSchedulerLock(s)
	if err != nil || lock == nil {
		t.Fatalf("初回のロック取得: %v", err)
	}
	checkRunning(true)
	other, err := tryAcquireSchedulerLock(s)
	if err != nil || other != nil {
		t.Fatalf("二重起動時のロック = %v, %v", other, err)
	}
	if err := lock.Close(); err != nil {
		t.Fatal(err)
	}
	checkRunning(false)
}
