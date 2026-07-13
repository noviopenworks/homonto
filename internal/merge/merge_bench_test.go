package merge

import "testing"

func BenchmarkMerge(b *testing.B) {
	base := []byte("alpha\nbeta\ngamma\ndelta\nepsilon\n")
	local := []byte("alpha\nBETA-local\ngamma\ndelta\nepsilon\n")
	upstream := []byte("alpha\nbeta\ngamma\nDELTA-upstream\nepsilon\n")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Merge(base, local, upstream)
	}
}
