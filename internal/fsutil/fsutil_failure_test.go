package fsutil

import (
	"os"
	"path/filepath"
	"testing"
)

// WriteAtomic must fail cleanly when the target directory cannot be created:
// return an error, change nothing, and leave no temp file behind. (Mode
// preservation, the 0600 default, and symlink write-through are covered by
// fsutil_test.go.)
func TestWriteAtomicDirCreateFailureIsClean(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("i am a file"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A path *under* a regular file: MkdirAll(blocker/sub) must fail.
	target := filepath.Join(blocker, "sub", "x.json")
	if err := WriteAtomic(target, []byte("data")); err == nil {
		t.Fatal("WriteAtomic must fail when the target directory cannot be created")
	}
	if b, _ := os.ReadFile(blocker); string(b) != "i am a file" {
		t.Fatalf("blocker file was modified: %q", b)
	}
	ents, _ := os.ReadDir(dir)
	for _, e := range ents {
		if e.Name() != "blocker" {
			t.Fatalf("unexpected leftover entry after a failed write: %s", e.Name())
		}
	}
}
