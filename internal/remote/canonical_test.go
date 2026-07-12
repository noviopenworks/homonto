package remote

import "testing"

func TestCanonicalDigestDeterministic(t *testing.T) {
	a := Tree{Files: []FileEntry{
		{Path: "a.txt", Mode: 0o644, Data: []byte("hello")},
		{Path: "b/c.txt", Mode: 0o644, Data: []byte("world")},
	}}
	// same content, reversed insertion order — must canonicalize identically
	b := Tree{Files: []FileEntry{
		{Path: "b/c.txt", Mode: 0o644, Data: []byte("world")},
		{Path: "a.txt", Mode: 0o644, Data: []byte("hello")},
	}}
	if CanonicalDigest(a).String() != CanonicalDigest(b).String() {
		t.Fatal("canonical digest must be independent of insertion order")
	}
}

func TestCanonicalDigestSensitivity(t *testing.T) {
	base := Tree{Files: []FileEntry{{Path: "a.txt", Mode: 0o644, Data: []byte("x")}}}
	d0 := CanonicalDigest(base).String()

	diffContent := Tree{Files: []FileEntry{{Path: "a.txt", Mode: 0o644, Data: []byte("y")}}}
	if CanonicalDigest(diffContent).String() == d0 {
		t.Error("different content must change the digest")
	}
	diffPath := Tree{Files: []FileEntry{{Path: "z.txt", Mode: 0o644, Data: []byte("x")}}}
	if CanonicalDigest(diffPath).String() == d0 {
		t.Error("different path must change the digest")
	}
	diffExec := Tree{Files: []FileEntry{{Path: "a.txt", Mode: 0o755, Data: []byte("x")}}}
	if CanonicalDigest(diffExec).String() == d0 {
		t.Error("the executable bit must change the digest")
	}
}

func TestCanonicalDigestEmptyTree(t *testing.T) {
	d := CanonicalDigest(Tree{})
	if d.IsZero() || d.Algo != "sha256" || len(d.Hex) != 64 {
		t.Fatalf("empty tree must have a well-defined sha256 digest, got %+v", d)
	}
}
