package remote

import "testing"

func FuzzParseDigest(f *testing.F) {
	f.Add("sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	f.Add("")
	f.Add("sha256:")
	f.Add("md5:abcd")
	f.Add("sha256:ZZZ")
	f.Add("remote:https://h.test/x")
	f.Fuzz(func(t *testing.T, s string) {
		d, err := ParseDigest(s)
		if err != nil {
			return
		}
		// A parsed digest must be well-formed and round-trip exactly.
		if d.Algo != "sha256" || len(d.Hex) != 64 {
			t.Fatalf("accepted malformed digest from %q: %+v", s, d)
		}
		if d.String() != s {
			t.Fatalf("round-trip mismatch: ParseDigest(%q).String() = %q", s, d.String())
		}
		if !d.Equal(d) {
			t.Fatalf("digest not equal to itself: %+v", d)
		}
	})
}
