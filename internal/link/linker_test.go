package link

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLinkCreatesAndIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "content", "skills", "graphify")
	os.MkdirAll(src, 0o755)
	dst := filepath.Join(dir, "claude", "skills", "graphify")

	changed, err := Link(src, dst)
	if err != nil || !changed {
		t.Fatalf("first link changed=%v err=%v", changed, err)
	}
	got, _ := os.Readlink(dst)
	if got != src {
		t.Fatalf("symlink points to %q", got)
	}
	changed, err = Link(src, dst)
	if err != nil || changed {
		t.Fatalf("second link should be no-op: changed=%v err=%v", changed, err)
	}
}

// TestLinkRelinksWrongTargetSymlink reproduces the verify round's dead end:
// status promised "will reset on apply" but Link returned a conflict for a
// symlink pointing elsewhere. Relinking a symlink destroys no data, so it
// must be repaired in place. (A regular file at dst stays a conflict.)
func TestLinkRelinksWrongTargetSymlink(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "content", "skills", "graphify")
	other := filepath.Join(dir, "elsewhere")
	os.MkdirAll(src, 0o755)
	os.MkdirAll(other, 0o755)
	dst := filepath.Join(dir, "claude", "skills", "graphify")
	os.MkdirAll(filepath.Dir(dst), 0o755)
	if err := os.Symlink(other, dst); err != nil {
		t.Fatal(err)
	}

	changed, err := Link(src, dst)
	if err != nil {
		t.Fatalf("wrong-target symlink must be relinked, got error: %v", err)
	}
	if !changed {
		t.Fatal("relink must report changed=true")
	}
	if got, _ := os.Readlink(dst); got != src {
		t.Fatalf("symlink points to %q, want %q", got, src)
	}
}

func TestLinkConflictDoesNotClobber(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	os.MkdirAll(src, 0o755)
	dst := filepath.Join(dir, "dst")
	os.WriteFile(dst, []byte("real file"), 0o644) // not a symlink

	_, err := Link(src, dst)
	if err == nil || !strings.Contains(err.Error(), "conflict") {
		t.Fatalf("expected conflict error, got %v", err)
	}
	if b, _ := os.ReadFile(dst); string(b) != "real file" {
		t.Fatal("conflict clobbered the real file")
	}
}
