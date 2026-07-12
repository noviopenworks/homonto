package remote

import (
	"encoding/json"
	"fmt"
	"os"
)

// Revocations is a set of banned content digests. Any resolve of a revoked
// digest fails closed, even on a warm cache.
type Revocations struct {
	set map[string]bool
}

// LoadRevocations reads a JSON array of "sha256:<hex>" strings from path. A
// missing file yields an empty (allow-all) list; a malformed digest entry fails
// closed so a corrupt revocation file cannot silently permit content.
func LoadRevocations(path string) (Revocations, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Revocations{set: map[string]bool{}}, nil
		}
		return Revocations{}, fmt.Errorf("remote: revocations: %w", err)
	}
	var entries []string
	if err := json.Unmarshal(data, &entries); err != nil {
		return Revocations{}, fmt.Errorf("remote: revocations %q: %w", path, err)
	}
	set := make(map[string]bool, len(entries))
	for _, e := range entries {
		d, err := ParseDigest(e)
		if err != nil {
			return Revocations{}, fmt.Errorf("remote: revocations %q: %w", path, err)
		}
		set[d.String()] = true
	}
	return Revocations{set: set}, nil
}

// Contains reports whether the digest is revoked.
func (r Revocations) Contains(d Digest) bool {
	return r.set[d.String()]
}
