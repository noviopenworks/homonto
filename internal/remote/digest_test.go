package remote

import "testing"

func TestParseDigest(t *testing.T) {
	valid := "sha256:" + "a" + "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcde"
	cases := []struct {
		name string
		in   string
		ok   bool
	}{
		{"valid lowercase", valid, true},
		{"blank", "", false},
		{"no algo prefix", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", false},
		{"wrong algo", "md5:0123456789abcdef0123456789abcdef", false},
		{"too short", "sha256:abcd", false},
		{"too long", "sha256:" + "a0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", false},
		{"non-hex", "sha256:zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz", false},
		{"uppercase hex rejected", "sha256:A123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDE", false},
		{"missing hex", "sha256:", false},
		{"trailing space", valid + " ", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d, err := ParseDigest(c.in)
			if c.ok && err != nil {
				t.Fatalf("ParseDigest(%q) unexpected error: %v", c.in, err)
			}
			if !c.ok && err == nil {
				t.Fatalf("ParseDigest(%q) expected error, got %+v", c.in, d)
			}
			if c.ok {
				if d.Algo != "sha256" {
					t.Errorf("algo = %q, want sha256", d.Algo)
				}
				if d.String() != c.in {
					t.Errorf("String() = %q, want %q", d.String(), c.in)
				}
			}
		})
	}
}

func TestDigestEqual(t *testing.T) {
	a, _ := ParseDigest("sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	b, _ := ParseDigest("sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	c, _ := ParseDigest("sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	if !a.Equal(b) {
		t.Error("identical digests should be equal")
	}
	if a.Equal(c) {
		t.Error("different digests should not be equal")
	}
}
