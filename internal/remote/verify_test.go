package remote

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// resolveFixture builds a file:// tar.gz source and returns its source + true pin.
func resolveFixture(t *testing.T) (RemoteSource, Digest) {
	t.Helper()
	dir := t.TempDir()
	p := writeFixtureTarGz(t, dir, "pkg.tar.gz")
	src, err := ParseRemoteSource("remote:file://" + p)
	if err != nil {
		t.Fatal(err)
	}
	tree, _, err := Fetch(context.Background(), src, DefaultLimits)
	if err != nil {
		t.Fatal(err)
	}
	return src, CanonicalDigest(tree)
}

func newResolver(t *testing.T) *Resolver {
	t.Helper()
	root := t.TempDir()
	return &Resolver{
		Cache:       &Cache{Root: filepath.Join(root, "cache")},
		Revocations: Revocations{},
		Limits:      DefaultLimits,
	}
}

func TestResolveHappyPath(t *testing.T) {
	src, pin := resolveFixture(t)
	r := newResolver(t)
	dir, err := r.Resolve(context.Background(), src, pin)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "agent.md")); err != nil {
		t.Fatalf("materialized content missing: %v", err)
	}
	if !r.Cache.Has(pin) {
		t.Fatal("verified content should be cached")
	}
}

func TestResolvePinMismatchFailsClosed(t *testing.T) {
	src, _ := resolveFixture(t)
	wrong, _ := ParseDigest("sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	r := newResolver(t)
	if _, err := r.Resolve(context.Background(), src, wrong); err == nil {
		t.Fatal("a pin mismatch must fail closed")
	}
	if r.Cache.Has(wrong) {
		t.Fatal("a mismatched resolve must not write a cache entry")
	}
}

func TestResolveRevokedFailsClosed(t *testing.T) {
	src, pin := resolveFixture(t)
	r := newResolver(t)
	r.Revocations = mustRevoke(t, pin)
	if _, err := r.Resolve(context.Background(), src, pin); err == nil {
		t.Fatal("a revoked pin must fail closed")
	}
	if r.Cache.Has(pin) {
		t.Fatal("a revoked resolve must not write a cache entry")
	}
}

func TestResolveRevokedEvenWhenCached(t *testing.T) {
	src, pin := resolveFixture(t)
	r := newResolver(t)
	// warm the cache with a clean resolve
	if _, err := r.Resolve(context.Background(), src, pin); err != nil {
		t.Fatal(err)
	}
	// now revoke; a subsequent resolve must fail even though it is cached
	r.Revocations = mustRevoke(t, pin)
	if _, err := r.Resolve(context.Background(), src, pin); err == nil {
		t.Fatal("revocation must be enforced on the cache-hit path")
	}
}

func TestResolveOfflineFromCache(t *testing.T) {
	src, pin := resolveFixture(t)
	r := newResolver(t)
	if _, err := r.Resolve(context.Background(), src, pin); err != nil {
		t.Fatal(err)
	}
	// Break the source so any fetch would fail; a warm cache must still resolve.
	broken := RemoteSource{URL: "file:///nonexistent/xyz.tar.gz", Transport: TransportFile}
	dir, err := r.ResolveCached(broken, pin)
	if err != nil {
		t.Fatalf("cached offline resolve should succeed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "agent.md")); err != nil {
		t.Fatalf("offline content missing: %v", err)
	}
}

func mustRevoke(t *testing.T, d Digest) Revocations {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "revoked.json")
	if err := os.WriteFile(path, []byte(`["`+d.String()+`"]`), 0o644); err != nil {
		t.Fatal(err)
	}
	rev, err := LoadRevocations(path)
	if err != nil {
		t.Fatal(err)
	}
	return rev
}
