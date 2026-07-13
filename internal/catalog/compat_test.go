package catalog

import "testing"

func TestSatisfiesLoose_StripsPreRelease(t *testing.T) {
	cases := []struct {
		v, c string
		want bool
	}{
		{"0.1.0-dev", ">=0.1.0", true}, // dev build satisfies the release constraint
		{"0.1.0-dev", ">=0.2.0", false},
		{"0.1.0+build.5", ">=0.1.0", true},
		{"0.2.0-rc1", ">=0.1.0", true},
		{"0.1.0", ">=0.1.0", true},
		{"0.0.9", ">=0.1.0", false},
	}
	for _, tc := range cases {
		got, err := SatisfiesLoose(tc.v, tc.c)
		if err != nil {
			t.Errorf("SatisfiesLoose(%q,%q) err=%v", tc.v, tc.c, err)
			continue
		}
		if got != tc.want {
			t.Errorf("SatisfiesLoose(%q,%q)=%v, want %v", tc.v, tc.c, got, tc.want)
		}
	}
}
