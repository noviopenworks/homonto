package agentblob

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/noviopenworks/homonto/internal/agentlock"
)

// Put then Get round-trips the content and the returned hash equals the
// lockfile content hash (agentlock.HashContent).
func TestPutGetRoundTrip(t *testing.T) {
	dir := t.TempDir()
	content := []byte("# agent\nhello world\n")

	hash, err := Put(dir, content)
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if hash != agentlock.HashContent(content) {
		t.Fatalf("Put hash = %s, want %s (agentlock.HashContent)", hash, agentlock.HashContent(content))
	}

	got, ok, err := Get(dir, hash)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !ok {
		t.Fatalf("Get ok = false, want true")
	}
	if string(got) != string(content) {
		t.Fatalf("Get content = %q, want %q", got, content)
	}
}

// Put is idempotent and content-addressed: two Puts of the same content return
// the same hash and leave a single blob file on disk.
func TestPutIdempotent(t *testing.T) {
	dir := t.TempDir()
	content := []byte("same content\n")

	h1, err := Put(dir, content)
	if err != nil {
		t.Fatalf("first Put: %v", err)
	}
	h2, err := Put(dir, content)
	if err != nil {
		t.Fatalf("second Put: %v", err)
	}
	if h1 != h2 {
		t.Fatalf("hashes differ: %s vs %s", h1, h2)
	}

	entries, err := os.ReadDir(filepath.Join(dir, "agents-blobs"))
	if err != nil {
		t.Fatalf("read blobs dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("blobs dir holds %d files, want 1", len(entries))
	}
}

// Get of a missing hash returns (nil, false, nil).
func TestGetMissing(t *testing.T) {
	dir := t.TempDir()
	got, ok, err := Get(dir, "0000000000000000000000000000000000000000000000000000000000000000")
	if err != nil {
		t.Fatalf("Get missing err = %v, want nil", err)
	}
	if ok {
		t.Fatalf("Get missing ok = true, want false")
	}
	if got != nil {
		t.Fatalf("Get missing content = %q, want nil", got)
	}
}

// The on-disk blob path is agents-blobs/<hash>.
func TestPutWritesExpectedPath(t *testing.T) {
	dir := t.TempDir()
	content := []byte("payload")
	hash, err := Put(dir, content)
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	p := filepath.Join(dir, "agents-blobs", hash)
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("expected blob at %s: %v", p, err)
	}
	if string(b) != string(content) {
		t.Fatalf("blob bytes = %q, want %q", b, content)
	}
}
