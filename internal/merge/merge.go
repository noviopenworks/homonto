// Package merge is a pure, dependency-free line-based three-way merge. Given a
// common base and two derived versions (local and upstream), it reproduces the
// changes each side made to disjoint regions of base and emits git-style
// conflict markers for regions both sides changed differently. The algorithm is
// deterministic and never silently mis-merges: it may conservatively conflict on
// some inputs a full diff3 could auto-merge, but a taken side is always exactly
// one side's content.
package merge

import "strings"

// Merge performs a line-based three-way merge of local and upstream against a
// common base, returning the merged bytes and the number of conflict regions
// emitted. Merge(x, x, x) returns x byte-for-byte with 0 conflicts.
func Merge(base, local, upstream []byte) (result []byte, conflicts int) {
	baseLines := splitLines(base)
	localLines := splitLines(local)
	upstreamLines := splitLines(upstream)

	commonL := lcsLineIndices(baseLines, localLines)
	commonU := lcsLineIndices(baseLines, upstreamLines)
	anchors := intersectAnchors(commonL, commonU)

	// Bracket the anchors with virtual start/end sentinels so every region of
	// the three inputs is covered by exactly one consecutive-anchor gap.
	walk := make([]anchor, 0, len(anchors)+2)
	walk = append(walk, anchor{b: -1, l: -1, u: -1})
	walk = append(walk, anchors...)
	walk = append(walk, anchor{b: len(baseLines), l: len(localLines), u: len(upstreamLines)})

	var out []string
	for k := 0; k+1 < len(walk); k++ {
		p, q := walk[k], walk[k+1]
		gapB := baseLines[p.b+1 : q.b]
		gapL := localLines[p.l+1 : q.l]
		gapU := upstreamLines[p.u+1 : q.u]

		switch {
		case equalLines(gapL, gapB):
			// local unchanged in this gap → take upstream.
			out = append(out, gapU...)
		case equalLines(gapU, gapB):
			// upstream unchanged in this gap → take local.
			out = append(out, gapL...)
		case equalLines(gapL, gapU):
			// both sides made the identical change → take it once.
			out = append(out, gapL...)
		default:
			out = append(out, "<<<<<<< local\n")
			out = append(out, gapL...)
			out = append(out, "=======\n")
			out = append(out, gapU...)
			out = append(out, ">>>>>>> source\n")
			conflicts++
		}

		// Emit the anchor line itself (equal across all three), unless q is the
		// end sentinel, which has no line to emit.
		if q.b != len(baseLines) {
			out = append(out, localLines[q.l])
		}
	}

	return []byte(strings.Join(out, "")), conflicts
}

// splitLines splits b into lines, each retaining its trailing "\n" so that
// joining with "" reproduces the input exactly, including whether it ended with
// a newline. Empty input yields no lines.
func splitLines(b []byte) []string {
	if len(b) == 0 {
		return nil
	}
	parts := strings.SplitAfter(string(b), "\n")
	// SplitAfter appends a trailing "" element iff the input ended with "\n";
	// drop it so the trailing-newline flag lives only in the last real line.
	if n := len(parts); n > 0 && parts[n-1] == "" {
		parts = parts[:n-1]
	}
	return parts
}

// pair is one aligned index match between two line slices: a's index and b's.
type pair struct{ a, b int }

// lcsLineIndices returns the aligned index pairs of a longest common
// subsequence of a and b under line equality, in increasing order of both
// indices. O(len(a)*len(b)) dynamic programming; agent files are small.
func lcsLineIndices(a, b []string) []pair {
	n, m := len(a), len(b)
	if n == 0 || m == 0 {
		return nil
	}
	// dp[i][j] = length of the LCS of a[i:] and b[j:].
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}
	var pairs []pair
	for i, j := 0, 0; i < n && j < m; {
		switch {
		case a[i] == b[j]:
			pairs = append(pairs, pair{a: i, b: j})
			i++
			j++
		case dp[i+1][j] >= dp[i][j+1]:
			i++
		default:
			j++
		}
	}
	return pairs
}

// anchor is a base line unchanged in all three inputs, carrying its matched
// local and upstream line indices.
type anchor struct{ b, l, u int }

// intersectAnchors returns the base indices present in both LCS results — the
// lines unchanged in local AND upstream — as anchors. Both inputs are increasing
// in base index (pair.a), so a two-pointer merge suffices.
func intersectAnchors(commonL, commonU []pair) []anchor {
	var out []anchor
	i, j := 0, 0
	for i < len(commonL) && j < len(commonU) {
		bi, bj := commonL[i].a, commonU[j].a
		switch {
		case bi == bj:
			out = append(out, anchor{b: bi, l: commonL[i].b, u: commonU[j].b})
			i++
			j++
		case bi < bj:
			i++
		default:
			j++
		}
	}
	return out
}

// equalLines reports whether two line slices are element-wise equal.
func equalLines(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
