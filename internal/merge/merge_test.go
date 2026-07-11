package merge

import (
	"strings"
	"testing"
)

// roundTrip: Merge(x,x,x) must return byte-identical x with 0 conflicts, for a
// range of trailing-newline shapes and sizes.
func TestMergeRoundTripIdentical(t *testing.T) {
	cases := map[string]string{
		"empty":                    "",
		"single line no newline":   "only",
		"single line with newline": "only\n",
		"multi trailing newline":   "a\nb\nc\n",
		"multi no trailing":        "a\nb\nc",
		"blank interior line":      "a\n\nc\n",
		"just newline":             "\n",
	}
	for name, x := range cases {
		t.Run(name, func(t *testing.T) {
			got, n := Merge([]byte(x), []byte(x), []byte(x))
			if string(got) != x {
				t.Fatalf("Merge(x,x,x) not byte-identical:\nwant %q\ngot  %q", x, got)
			}
			if n != 0 {
				t.Fatalf("Merge(x,x,x) conflicts = %d, want 0", n)
			}
		})
	}
}

// only local changed: upstream == base, result == local, 0 conflicts.
func TestMergeOnlyLocalChanged(t *testing.T) {
	base := "a\nb\nc\nd\n"
	local := "a\nB\nc\nd\n"
	got, n := Merge([]byte(base), []byte(local), []byte(base))
	if string(got) != local {
		t.Fatalf("Merge(b,l,b) = %q, want local %q", got, local)
	}
	if n != 0 {
		t.Fatalf("conflicts = %d, want 0", n)
	}
}

// only upstream changed: local == base, result == upstream, 0 conflicts.
func TestMergeOnlyUpstreamChanged(t *testing.T) {
	base := "a\nb\nc\nd\n"
	upstream := "a\nb\nC\nd\n"
	got, n := Merge([]byte(base), []byte(base), []byte(upstream))
	if string(got) != upstream {
		t.Fatalf("Merge(b,b,u) = %q, want upstream %q", got, upstream)
	}
	if n != 0 {
		t.Fatalf("conflicts = %d, want 0", n)
	}
}

// disjoint edits: local edits an early line, upstream a later line of a 6-line
// base; both edits present with 0 conflicts.
func TestMergeDisjointEditsAutoMerge(t *testing.T) {
	base := "a\nb\nc\nd\ne\nf\n"
	local := "A\nb\nc\nd\ne\nf\n"    // line 1 edited
	upstream := "a\nb\nc\nd\nE\nf\n" // line 5 edited
	got, n := Merge([]byte(base), []byte(local), []byte(upstream))
	want := "A\nb\nc\nd\nE\nf\n"
	if string(got) != want {
		t.Fatalf("disjoint merge = %q, want %q", got, want)
	}
	if n != 0 {
		t.Fatalf("conflicts = %d, want 0", n)
	}
}

// insertions at opposite ends auto-merge with 0 conflicts.
func TestMergeInsertionsAtBothEndsAutoMerge(t *testing.T) {
	base := "a\nb\nc\n"
	local := "X\na\nb\nc\n"    // prepend
	upstream := "a\nb\nc\nY\n" // append
	got, n := Merge([]byte(base), []byte(local), []byte(upstream))
	want := "X\na\nb\nc\nY\n"
	if string(got) != want {
		t.Fatalf("insertion merge = %q, want %q", got, want)
	}
	if n != 0 {
		t.Fatalf("conflicts = %d, want 0", n)
	}
}

// overlapping edits: both change the same line differently → a conflict block
// with all three markers and conflicts >= 1.
func TestMergeOverlappingEditsConflict(t *testing.T) {
	base := "a\nb\nc\nd\n"
	local := "a\nb\nL3\nd\n"
	upstream := "a\nb\nU3\nd\n"
	got, n := Merge([]byte(base), []byte(local), []byte(upstream))
	if n < 1 {
		t.Fatalf("overlapping edits conflicts = %d, want >= 1", n)
	}
	s := string(got)
	for _, marker := range []string{"<<<<<<< local", "=======", ">>>>>>> source"} {
		if !strings.Contains(s, marker) {
			t.Fatalf("result missing marker %q:\n%s", marker, s)
		}
	}
	if !strings.Contains(s, "L3") || !strings.Contains(s, "U3") {
		t.Fatalf("conflict must contain both sides:\n%s", s)
	}
	// exact bytes for the conflict shape.
	want := "a\nb\n<<<<<<< local\nL3\n=======\nU3\n>>>>>>> source\nd\n"
	if s != want {
		t.Fatalf("conflict bytes:\nwant %q\ngot  %q", want, s)
	}
}

// identical edit on both sides: taken once, no markers, 0 conflicts.
func TestMergeIdenticalEditBothSides(t *testing.T) {
	base := "a\nb\nc\nd\n"
	edit := "a\nb\nX3\nd\n"
	got, n := Merge([]byte(base), []byte(edit), []byte(edit))
	if n != 0 {
		t.Fatalf("identical edit conflicts = %d, want 0", n)
	}
	if string(got) != edit {
		t.Fatalf("identical edit = %q, want %q", got, edit)
	}
	if strings.Contains(string(got), "<<<<<<<") {
		t.Fatalf("identical edit must not produce markers:\n%s", got)
	}
	// change appears exactly once, not duplicated.
	if strings.Count(string(got), "X3") != 1 {
		t.Fatalf("identical edit must appear once, got %d:\n%s", strings.Count(string(got), "X3"), got)
	}
}

// adjacent edits on different lines with no unchanged line between them fall in
// one gap and conservatively conflict (documented safe over-conflict).
func TestMergeAdjacentEditsConflict(t *testing.T) {
	base := "a\nb\nc\nd\ne\n"
	local := "a\nB\nc\nd\ne\n"    // line 2
	upstream := "a\nb\nC\nd\ne\n" // line 3 (adjacent)
	got, n := Merge([]byte(base), []byte(local), []byte(upstream))
	if n < 1 {
		t.Fatalf("adjacent edits conflicts = %d, want >= 1 (conservative), got %q", n, got)
	}
	for _, marker := range []string{"<<<<<<< local", "=======", ">>>>>>> source"} {
		if !strings.Contains(string(got), marker) {
			t.Fatalf("adjacent conflict missing marker %q:\n%s", marker, got)
		}
	}
}

// table-driven auto-merge / passthrough cases with expected exact output.
func TestMergeTable(t *testing.T) {
	cases := []struct {
		name              string
		base, local, upst string
		want              string
		conflicts         int
	}{
		{"all empty", "", "", "", "", 0},
		{"local deletes line", "a\nb\nc\n", "a\nc\n", "a\nb\nc\n", "a\nc\n", 0},
		{"upstream deletes line", "a\nb\nc\n", "a\nb\nc\n", "a\nc\n", "a\nc\n", 0},
		{"both delete same line", "a\nb\nc\n", "a\nc\n", "a\nc\n", "a\nc\n", 0},
		{"disjoint no trailing nl", "a\nb\nc", "A\nb\nc", "a\nb\nC", "A\nb\nC", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, n := Merge([]byte(tc.base), []byte(tc.local), []byte(tc.upst))
			if string(got) != tc.want {
				t.Fatalf("result:\nwant %q\ngot  %q", tc.want, got)
			}
			if n != tc.conflicts {
				t.Fatalf("conflicts = %d, want %d", n, tc.conflicts)
			}
		})
	}
}
