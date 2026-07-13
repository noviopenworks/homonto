package applylock

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAcquireReleaseLeavesNoStaleLock: a normal acquire creates the lockfile and
// release removes it, leaving nothing behind.
func TestAcquireReleaseLeavesNoStaleLock(t *testing.T) {
	dir := t.TempDir()
	lk, err := Acquire(dir)
	if err != nil {
		t.Fatalf("first acquire failed: %v", err)
	}
	lockPath := filepath.Join(dir, "apply.lock")
	if _, err := os.Stat(lockPath); err != nil {
		t.Fatalf("lockfile should exist while held: %v", err)
	}
	if err := lk.Release(); err != nil {
		t.Fatalf("release failed: %v", err)
	}
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatalf("lockfile should be gone after release, stat err = %v", err)
	}
	// Re-acquire after release must succeed (no stale lock).
	lk2, err := Acquire(dir)
	if err != nil {
		t.Fatalf("re-acquire after release failed: %v", err)
	}
	_ = lk2.Release()
}

// TestSecondAcquireFailsFast: while a lock is held, a second acquire on the same
// project fails fast with a clear message and does not disturb the held lock.
func TestSecondAcquireFailsFast(t *testing.T) {
	dir := t.TempDir()
	lk, err := Acquire(dir)
	if err != nil {
		t.Fatalf("first acquire failed: %v", err)
	}
	defer lk.Release()

	_, err = Acquire(dir)
	if err == nil {
		t.Fatal("second acquire should fail while the lock is held")
	}
	if !strings.Contains(err.Error(), "another apply is in progress") {
		t.Fatalf("error should mention another apply in progress, got: %v", err)
	}
	// The original lockfile must still be present and owned by the first holder.
	if _, serr := os.Stat(filepath.Join(dir, "apply.lock")); serr != nil {
		t.Fatalf("held lockfile disturbed by a failed acquire: %v", serr)
	}
}
