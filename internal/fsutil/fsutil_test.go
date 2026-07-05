package fsutil

import (
	"os"
	"path/filepath"
	"testing"
)

// TestWriteAtomicPreservesExistingMode: a 0600 config file (it may hold
// resolved secrets) must never be loosened by a rewrite.
func TestWriteAtomicPreservesExistingMode(t *testing.T) {
	p := filepath.Join(t.TempDir(), "cfg.json")
	if err := os.WriteFile(p, []byte(`{"old":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := WriteAtomic(p, []byte(`{"new":true}`)); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if got := fi.Mode().Perm(); got != 0o600 {
		t.Fatalf("existing 0600 file loosened to %v after write", got)
	}
	if b, _ := os.ReadFile(p); string(b) != `{"new":true}` {
		t.Fatalf("content = %s", b)
	}
}

// TestWriteAtomicWritesThroughSymlink reproduces the verify round's dotfiles
// finding: rename-over-path replaces a symlinked target (~/.claude.json ->
// dotfiles/claude.json) with a regular file, silently diverging from the
// dotfiles copy. The write must land in the link's target, keeping the link.
func TestWriteAtomicWritesThroughSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "dotfiles", "claude.json")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte(`{"old":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	lnk := filepath.Join(dir, ".claude.json")
	if err := os.Symlink(target, lnk); err != nil {
		t.Fatal(err)
	}
	if err := WriteAtomic(lnk, []byte(`{"new":true}`)); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Lstat(lnk)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Fatal("symlink was replaced by a regular file")
	}
	if b, _ := os.ReadFile(target); string(b) != `{"new":true}` {
		t.Fatalf("link target content = %s, want the new content", b)
	}
}

// TestWriteAtomicNewFileIs0600: files we create may receive resolved secrets
// on a later apply, so the safe default is owner-only.
func TestWriteAtomicNewFileIs0600(t *testing.T) {
	p := filepath.Join(t.TempDir(), "sub", "cfg.json")
	if err := WriteAtomic(p, []byte(`{}`)); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if got := fi.Mode().Perm(); got != 0o600 {
		t.Fatalf("new file created %v, want 0600", got)
	}
}
