package remote

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRevocations(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "revoked.json")

	// Missing file → empty list, no error.
	rev, err := LoadRevocations(path)
	if err != nil {
		t.Fatalf("missing revocation file must not error: %v", err)
	}
	d, _ := ParseDigest("sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	if rev.Contains(d) {
		t.Fatal("empty list should contain nothing")
	}

	if err := os.WriteFile(path, []byte(`["sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"]`), 0o644); err != nil {
		t.Fatal(err)
	}
	rev, err = LoadRevocations(path)
	if err != nil {
		t.Fatal(err)
	}
	if !rev.Contains(d) {
		t.Fatal("listed digest must be reported revoked")
	}
	other, _ := ParseDigest("sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	if rev.Contains(other) {
		t.Fatal("unlisted digest must not be revoked")
	}
}

func TestLoadRevocationsRejectsMalformed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "revoked.json")
	if err := os.WriteFile(path, []byte(`["not-a-digest"]`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadRevocations(path); err == nil {
		t.Fatal("a malformed digest entry must fail closed")
	}
}
