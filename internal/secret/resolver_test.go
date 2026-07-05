package secret

import (
	"strings"
	"testing"
)

func TestResolveEnvAndPass(t *testing.T) {
	r := &Resolver{
		Getenv: func(k string) string {
			if k == "FOO" {
				return "envval"
			}
			return ""
		},
		Pass: func(p string) (string, error) {
			if p == "ai/brave" {
				return "passval", nil
			}
			return "", &notFound{p}
		},
	}
	got, err := r.Resolve("a=${FOO} b=${pass:ai/brave}")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got != "a=envval b=passval" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveMissingEnvErrors(t *testing.T) {
	r := &Resolver{Getenv: func(string) string { return "" }, Pass: func(string) (string, error) { return "", nil }}
	_, err := r.Resolve("${MISSING}")
	if err == nil || !strings.Contains(err.Error(), "MISSING") {
		t.Fatalf("expected missing-env error, got %v", err)
	}
}

func TestContainsRef(t *testing.T) {
	if !ContainsRef("x ${Y}") || ContainsRef("plain") {
		t.Fatal("ContainsRef wrong")
	}
}

// TestResolveJSONMemoizesPassLookups: engine pre-resolves for validation and
// adapters resolve again on apply — each distinct token must hit pass once.
func TestResolveJSONMemoizesPassLookups(t *testing.T) {
	calls := 0
	r := &Resolver{
		Getenv: func(string) string { return "" },
		Pass: func(p string) (string, error) {
			calls++
			return "v", nil
		},
	}
	if _, err := r.ResolveJSON(`{"a":"${pass:ai/brave}","b":"${pass:ai/brave}"}`); err != nil {
		t.Fatal(err)
	}
	if _, err := r.ResolveJSON(`"${pass:ai/brave}"`); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("pass invoked %d times for one token across two resolves, want 1 (memoized)", calls)
	}
}

type notFound struct{ p string }

func (e *notFound) Error() string { return "not found: " + e.p }
