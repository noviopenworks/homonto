package remote

import (
	"context"
	"fmt"
)

// Resolver turns a pinned remote source into a local, verified cache directory.
// Its single invariant: no cache entry (and thus no downstream target file) is
// written until fetched content is validated, its canonical digest matches the
// pin, and the pin is not revoked.
type Resolver struct {
	Cache       *Cache
	Revocations Revocations
	Limits      Limits
}

// Resolve returns a local directory containing the pinned content, fetching and
// verifying it if not already cached. The pipeline order is:
//
//	cache hit → revocation check → return   (offline path)
//	otherwise: fetch → validate → canonical digest → pin match → revocation → cache
//
// Every failure aborts before any cache write, so malformed, tampered, or
// revoked content fails closed.
func (r *Resolver) Resolve(ctx context.Context, src RemoteSource, pin Digest) (string, error) {
	if r.Cache.Has(pin) {
		return r.ResolveCached(src, pin)
	}
	tree, _, err := Fetch(ctx, src, r.limits())
	if err != nil {
		return "", err
	}
	got := CanonicalDigest(tree)
	if !got.Equal(pin) {
		return "", fmt.Errorf("remote: pin mismatch for %q: declared %s but content is %s", RedactLocator(src.URL), pin, got)
	}
	if r.Revocations.Contains(got) {
		return "", fmt.Errorf("remote: content %s is revoked", got)
	}
	dir, err := r.Cache.Put(got, tree)
	if err != nil {
		return "", err
	}
	return dir, nil
}

// ResolveCached returns the cached directory for a pin without any network
// access. It re-hashes the cached content against the pin (so a locally
// tampered or corrupted cache entry fails closed) and enforces revocation. It
// errors if the pin is not cached.
func (r *Resolver) ResolveCached(_ RemoteSource, pin Digest) (string, error) {
	if !r.Cache.Has(pin) {
		return "", fmt.Errorf("remote: %s is not cached", pin)
	}
	if r.Revocations.Contains(pin) {
		return "", fmt.Errorf("remote: content %s is revoked", pin)
	}
	dir := r.Cache.Dir(pin)
	tree, _, err := treeFromDir(dir, r.limits())
	if err != nil {
		return "", fmt.Errorf("remote: reading cached %s: %w", pin, err)
	}
	if got := CanonicalDigest(tree); !got.Equal(pin) {
		return "", fmt.Errorf("remote: cached content for %s is corrupt or tampered (recomputed %s)", pin, got)
	}
	return dir, nil
}

func (r *Resolver) limits() Limits {
	if r.Limits == (Limits{}) {
		return DefaultLimits
	}
	return r.Limits
}
