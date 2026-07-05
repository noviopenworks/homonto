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
