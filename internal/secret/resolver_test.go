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

type notFound struct{ p string }

func (e *notFound) Error() string { return "not found: " + e.p }
