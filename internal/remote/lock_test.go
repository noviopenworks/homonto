package remote

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLockRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "remote.lock.json")

	// Missing file → empty lock, no error.
	l, err := LoadLock(path)
	if err != nil {
		t.Fatalf("missing lock must not error: %v", err)
	}
	if len(l.Entries) != 0 {
		t.Fatal("missing lock should be empty")
	}

	l.Set(LockEntry{Kind: "subagent", Name: "x", Locator: "https://h/x.tgz", Transport: "https", Digest: "sha256:aa", Size: 42})
	l.Set(LockEntry{Kind: "skill", Name: "y", Locator: "file:///y", Transport: "file", Digest: "sha256:bb", Size: 7})
	if err := l.Save(path); err != nil {
		t.Fatal(err)
	}

	l2, err := LoadLock(path)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := l2.Get("subagent", "x")
	if !ok || got.Digest != "sha256:aa" || got.Size != 42 {
		t.Fatalf("round-trip lost entry: %+v ok=%v", got, ok)
	}
}

func TestLockDiffStable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "remote.lock.json")
	l := Lock{}
	// insert in non-sorted order to prove the output is canonicalized
	l.Set(LockEntry{Kind: "subagent", Name: "z", Digest: "sha256:zz"})
	l.Set(LockEntry{Kind: "subagent", Name: "a", Digest: "sha256:aa"})
	if err := l.Save(path); err != nil {
		t.Fatal(err)
	}
	first, _ := os.ReadFile(path)

	l2, _ := LoadLock(path)
	if err := l2.Save(path); err != nil {
		t.Fatal(err)
	}
	second, _ := os.ReadFile(path)
	if string(first) != string(second) {
		t.Fatalf("lock must be byte-stable across saves:\n%s\n---\n%s", first, second)
	}
}

func TestLockRemove(t *testing.T) {
	l := Lock{}
	l.Set(LockEntry{Kind: "subagent", Name: "x", Digest: "sha256:aa"})
	l.Remove("subagent", "x")
	if _, ok := l.Get("subagent", "x"); ok {
		t.Fatal("removed entry should be gone")
	}
}

func TestLockDigests(t *testing.T) {
	l := Lock{}
	l.Set(LockEntry{Kind: "subagent", Name: "x", Digest: "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"})
	ds := l.Digests()
	if len(ds) != 1 || ds[0].Hex != "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" {
		t.Fatalf("Digests() = %+v", ds)
	}
}
