package link

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLinkCreatesAndIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	content := filepath.Join(dir, "content")
	src := filepath.Join(content, "skills", "graphify")
	os.MkdirAll(src, 0o755)
	dst := filepath.Join(dir, "claude", "skills", "graphify")

	changed, err := Link(src, dst, content)
	if err != nil || !changed {
		t.Fatalf("first link changed=%v err=%v", changed, err)
	}
	got, _ := os.Readlink(dst)
	if got != src {
		t.Fatalf("symlink points to %q", got)
	}
	changed, err = Link(src, dst, content)
	if err != nil || changed {
		t.Fatalf("second link should be no-op: changed=%v err=%v", changed, err)
	}
}

// TestLinkRelinksManagedWrongTarget: a symlink already pointing inside the
// managed content root is one of ours (e.g. a stale target from an older
// layout), so it is repaired in place — relinking one of our own symlinks
// destroys no user data.
func TestLinkRelinksManagedWrongTarget(t *testing.T) {
	dir := t.TempDir()
	content := filepath.Join(dir, "content")
	src := filepath.Join(content, "skills", "graphify")
	stale := filepath.Join(content, "skills", "graphify-old")
	os.MkdirAll(src, 0o755)
	os.MkdirAll(stale, 0o755)
	dst := filepath.Join(dir, "claude", "skills", "graphify")
	os.MkdirAll(filepath.Dir(dst), 0o755)
	if err := os.Symlink(stale, dst); err != nil {
		t.Fatal(err)
	}

	changed, err := Link(src, dst, content)
	if err != nil {
		t.Fatalf("managed wrong-target symlink must be relinked, got error: %v", err)
	}
	if !changed {
		t.Fatal("relink must report changed=true")
	}
	if got, _ := os.Readlink(dst); got != src {
		t.Fatalf("symlink points to %q, want %q", got, src)
	}
}

// TestLinkForeignSymlinkIsConflict: a symlink pointing OUTSIDE the managed
// content root is user-owned (e.g. a skill the user linked from their own
// dotfiles). homonto must treat it as a conflict and leave it untouched — never
// silently repoint or remove what it does not own.
func TestLinkForeignSymlinkIsConflict(t *testing.T) {
	dir := t.TempDir()
	content := filepath.Join(dir, "content")
	src := filepath.Join(content, "skills", "graphify")
	foreign := filepath.Join(dir, "elsewhere")
	os.MkdirAll(src, 0o755)
	os.MkdirAll(foreign, 0o755)
	dst := filepath.Join(dir, "claude", "skills", "graphify")
	os.MkdirAll(filepath.Dir(dst), 0o755)
	if err := os.Symlink(foreign, dst); err != nil {
		t.Fatal(err)
	}

	changed, err := Link(src, dst, content)
	if err == nil || !strings.Contains(err.Error(), "conflict") {
		t.Fatalf("foreign symlink must be a conflict, got changed=%v err=%v", changed, err)
	}
	if changed {
		t.Fatal("conflict must report changed=false")
	}
	if got, _ := os.Readlink(dst); got != foreign {
		t.Fatalf("conflict changed the foreign symlink: now points to %q, want %q", got, foreign)
	}
}

func TestLinkConflictDoesNotClobber(t *testing.T) {
	dir := t.TempDir()
	content := filepath.Join(dir, "content")
	src := filepath.Join(content, "skills", "graphify")
	os.MkdirAll(src, 0o755)
	dst := filepath.Join(dir, "dst")
	os.WriteFile(dst, []byte("real file"), 0o644) // not a symlink

	_, err := Link(src, dst, content)
	if err == nil || !strings.Contains(err.Error(), "conflict") {
		t.Fatalf("expected conflict error, got %v", err)
	}
	if b, _ := os.ReadFile(dst); string(b) != "real file" {
		t.Fatal("conflict clobbered the real file")
	}
}

// TestPlanForeignSymlinkIsConflict: Plan (used by status/plan rendering and the
// fail-fast in Apply) must also reject a symlink pointing outside content,
// rather than rendering it as a relink that Apply would refuse anyway.
func TestPlanForeignSymlinkIsConflict(t *testing.T) {
	dir := t.TempDir()
	content := filepath.Join(dir, "content")
	src := filepath.Join(content, "skills", "graphify")
	foreign := filepath.Join(dir, "elsewhere")
	os.MkdirAll(src, 0o755)
	os.MkdirAll(foreign, 0o755)
	dst := filepath.Join(dir, "claude", "skills", "graphify")
	os.MkdirAll(filepath.Dir(dst), 0o755)
	if err := os.Symlink(foreign, dst); err != nil {
		t.Fatal(err)
	}

	_, err := Plan(map[string]string{dst: src}, content)
	if err == nil || !strings.Contains(err.Error(), "conflict") {
		t.Fatalf("Plan must report conflict for foreign symlink, got %v", err)
	}
	if got, _ := os.Readlink(dst); got != foreign {
		t.Fatalf("Plan changed the foreign symlink: now points to %q", got)
	}
}
