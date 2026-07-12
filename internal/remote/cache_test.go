package remote

import (
	"os"
	"path/filepath"
	"testing"
)

func sampleTree() Tree {
	return Tree{Files: []FileEntry{
		{Path: "agent.md", Mode: 0o644, Data: []byte("# agent")},
		{Path: "ref/n.md", Mode: 0o644, Data: []byte("notes")},
	}}
}

func TestCachePutHasDir(t *testing.T) {
	c := &Cache{Root: t.TempDir()}
	tree := sampleTree()
	dg := CanonicalDigest(tree)

	if c.Has(dg) {
		t.Fatal("empty cache should not have the digest")
	}
	dir, err := c.Put(dg, tree)
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	if !c.Has(dg) {
		t.Fatal("cache should have the digest after put")
	}
	if dir != c.Dir(dg) {
		t.Fatalf("Put dir %q != Dir %q", dir, c.Dir(dg))
	}
	// materialized content present and correct
	got, err := os.ReadFile(filepath.Join(dir, "ref", "n.md"))
	if err != nil {
		t.Fatalf("read materialized: %v", err)
	}
	if string(got) != "notes" {
		t.Fatalf("content mismatch: %q", got)
	}
}

func TestCachePutIdempotentSamePath(t *testing.T) {
	c := &Cache{Root: t.TempDir()}
	tree := sampleTree()
	dg := CanonicalDigest(tree)
	d1, err := c.Put(dg, tree)
	if err != nil {
		t.Fatal(err)
	}
	d2, err := c.Put(dg, tree) // second put is a no-op returning the same path
	if err != nil {
		t.Fatal(err)
	}
	if d1 != d2 {
		t.Fatalf("same content must map to the same cache path: %q vs %q", d1, d2)
	}
}

func TestCacheGC(t *testing.T) {
	c := &Cache{Root: t.TempDir()}
	keep := sampleTree()
	drop := Tree{Files: []FileEntry{{Path: "x", Mode: 0o644, Data: []byte("other")}}}
	kd := CanonicalDigest(keep)
	dd := CanonicalDigest(drop)
	if _, err := c.Put(kd, keep); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Put(dd, drop); err != nil {
		t.Fatal(err)
	}

	// dry-run reclaims nothing but reports the unreferenced digest
	reclaimed, err := c.GC([]Digest{kd}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(reclaimed) != 1 || !reclaimed[0].Equal(dd) {
		t.Fatalf("dry-run should report dropping %s, got %+v", dd, reclaimed)
	}
	if !c.Has(dd) {
		t.Fatal("dry-run must not delete anything")
	}

	// real GC removes only the unreferenced digest
	reclaimed, err = c.GC([]Digest{kd}, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(reclaimed) != 1 || !reclaimed[0].Equal(dd) {
		t.Fatalf("gc should reclaim %s, got %+v", dd, reclaimed)
	}
	if c.Has(dd) {
		t.Fatal("unreferenced digest should be gone")
	}
	if !c.Has(kd) {
		t.Fatal("referenced digest must remain")
	}
}
