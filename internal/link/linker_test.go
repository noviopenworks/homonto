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
