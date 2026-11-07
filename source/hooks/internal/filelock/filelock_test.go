package filelock_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fza/sdd-helpers/source/hooks/internal/filelock"
)

func TestExclusiveWaitsForTheHolder(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".lock")

	holder, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatalf("opening the lock: %v", err)
	}
	defer holder.Close()

	waiter, err := os.OpenFile(path, os.O_RDWR, 0o600)
	if err != nil {
		t.Fatalf("opening the lock again: %v", err)
	}
	defer waiter.Close()

	err = filelock.Exclusive(holder)
	if err != nil {
		t.Fatalf("taking the free lock: %v", err)
	}

	taken := make(chan error, 1)

	go func() {
		taken <- filelock.Exclusive(waiter)
	}()

	select {
	case <-taken:
		t.Fatal("a held lock should keep a second taker waiting")
	case <-time.After(100 * time.Millisecond):
	}

	err = filelock.Unlock(holder)
	if err != nil {
		t.Fatalf("releasing the lock: %v", err)
	}

	select {
	case err = <-taken:
		if err != nil {
			t.Errorf("the waiter should take the lock once it is released, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("the waiter should take the lock once it is released")
	}
}

func TestTryExclusiveGivesWayToAHolder(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".lock")

	holder, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatalf("opening the lock: %v", err)
	}
	defer holder.Close()

	trier, err := os.OpenFile(path, os.O_RDWR, 0o600)
	if err != nil {
		t.Fatalf("opening the lock again: %v", err)
	}
	defer trier.Close()

	err = filelock.Exclusive(holder)
	if err != nil {
		t.Fatalf("taking the free lock: %v", err)
	}

	held, err := filelock.TryExclusive(trier)
	if err != nil || held {
		t.Errorf("a try should answer at once that somebody holds the lock, got %v %v", held, err)
	}

	err = filelock.Unlock(holder)
	if err != nil {
		t.Fatalf("releasing the lock: %v", err)
	}

	held, err = filelock.TryExclusive(trier)
	if err != nil || !held {
		t.Errorf("a try should take a lock nobody holds, got %v %v", held, err)
	}
}
