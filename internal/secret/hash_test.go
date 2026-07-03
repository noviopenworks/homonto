package secret

import (
	"regexp"
	"testing"
)

func TestHashStableAndHex(t *testing.T) {
	h1 := Hash("secret-value")
	h2 := Hash("secret-value")
	if h1 != h2 {
		t.Fatalf("hash not stable: %q vs %q", h1, h2)
	}
	if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(h1) {
		t.Fatalf("not lowercase hex sha256: %q", h1)
	}
}

func TestHashDiffersByInput(t *testing.T) {
	if Hash("a") == Hash("b") {
		t.Fatal("different inputs hashed equal")
	}
}

func TestHashNeverContainsInput(t *testing.T) {
	secretVal := "sk-super-secret-123"
	if got := Hash(secretVal); regexpContains(got, secretVal) {
		t.Fatalf("hash leaked the input: %q", got)
	}
}

func regexpContains(hay, needle string) bool {
	return len(needle) > 0 && len(hay) >= len(needle) && (indexOf(hay, needle) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
