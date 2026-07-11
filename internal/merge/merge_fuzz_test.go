package merge

import (
	"bytes"
	"testing"
)

// FuzzMerge exercises the three-way merge's core invariants on arbitrary inputs.
// It must never panic, and the identity / one-sided-change properties must hold
// exactly for every input.
func FuzzMerge(f *testing.F) {
	f.Add([]byte("a\nb\nc\n"), []byte("a\nb\nc\n"), []byte("a\nb\nc\n"))
	f.Add([]byte("a\nb\nc\n"), []byte("X\nb\nc\n"), []byte("a\nb\nY\n"))
	f.Add([]byte("a\nb\nc\n"), []byte("a\nZ\nc\n"), []byte("a\nW\nc\n"))
	f.Add([]byte(""), []byte("x"), []byte("y"))
	f.Add([]byte("l1\nl2"), []byte("l1"), []byte("l2\nl3\n"))

	f.Fuzz(func(t *testing.T, base, local, upstream []byte) {
		_, conflicts := Merge(base, local, upstream)
		if conflicts < 0 {
			t.Fatalf("negative conflict count %d", conflicts)
		}

		// Identity: Merge(x, x, x) == x with 0 conflicts.
		if got, c := Merge(base, base, base); c != 0 || !bytes.Equal(got, base) {
			t.Fatalf("Merge(x,x,x) must equal x with 0 conflicts: %d conflicts, %q vs %q", c, got, base)
		}
		// Upstream == base (only local changed) → take local exactly, no conflict.
		if got, c := Merge(base, local, base); c != 0 || !bytes.Equal(got, local) {
			t.Fatalf("Merge(base, local, base) must equal local with 0 conflicts: %d conflicts, %q vs %q", c, got, local)
		}
		// Local == base (only upstream changed) → take upstream exactly, no conflict.
		if got, c := Merge(base, base, upstream); c != 0 || !bytes.Equal(got, upstream) {
			t.Fatalf("Merge(base, base, upstream) must equal upstream with 0 conflicts: %d conflicts, %q vs %q", c, got, upstream)
		}
	})
}
