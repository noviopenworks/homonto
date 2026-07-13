package remote

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRedactLocatorStripsCredentials(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"userinfo", "https://user:s3cret@host/repo"},
		{"userinfo-token-only", "https://s3cret@host/repo.git"},
		{"query-token", "https://host/repo?token=s3cret"},
		{"query-access-token", "https://host/repo?ref=main&access_token=s3cret"},
		{"git-scheme-userinfo", "git+https://user:s3cret@host/repo#v1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := RedactLocator(c.in)
			if strings.Contains(got, "s3cret") {
				t.Fatalf("redacted locator still contains the secret: %q", got)
			}
			// The host must survive redaction so the locator stays meaningful.
			if !strings.Contains(got, "host") {
				t.Fatalf("redaction dropped the host: %q", got)
			}
		})
	}
}

// A locator with embedded credentials must not reach the lockfile verbatim.
func TestLockDoesNotPersistCredentials(t *testing.T) {
	path := filepath.Join(t.TempDir(), "remote.lock.json")
	l := Lock{}
	l.Set(LockEntry{
		Kind:      "subagent",
		Name:      "x",
		Locator:   "https://user:s3cret@host/repo?token=s3cret",
		Transport: "https",
		Digest:    "sha256:aa",
	})
	if err := l.Save(path); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), "s3cret") {
		t.Fatalf("lockfile leaked the credential:\n%s", data)
	}
	// The stored locator is the redacted form, retrievable and host-bearing.
	got, ok := l.Get("subagent", "x")
	if !ok || strings.Contains(got.Locator, "s3cret") {
		t.Fatalf("stored locator not redacted: %+v", got)
	}
}

// An emitted error surfacing a locator must not carry the secret.
func TestParseRemoteSourceErrorRedactsCredentials(t *testing.T) {
	// An unsupported scheme surfaces the URL in the error message.
	_, err := ParseRemoteSource("remote:ssh://user:s3cret@host/repo")
	if err == nil {
		t.Fatal("expected an error for an unsupported scheme")
	}
	if strings.Contains(err.Error(), "s3cret") {
		t.Fatalf("error leaked the credential: %v", err)
	}
}
