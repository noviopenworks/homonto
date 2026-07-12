// Package remote implements homonto's remote-source trust boundary: pinned,
// verified, fail-closed installation of remote resources. Its single guarantee
// is that no target file is mutated until fetched content is pinned-verified and
// passes every structural safety check.
package remote

import (
	"crypto/sha256"
	"fmt"
)

// Digest is a content pin. Only sha256 is accepted in the first increment.
type Digest struct {
	Algo string // always "sha256"
	Hex  string // 64 lowercase hex chars
}

// ParseDigest parses a "sha256:<64 lowercase hex>" pin. Anything else — a
// missing prefix, a non-sha256 algorithm, a wrong length, uppercase, or a
// non-hex character — is rejected so an invalid pin fails closed at load.
func ParseDigest(s string) (Digest, error) {
	const prefix = "sha256:"
	if len(s) <= len(prefix) || s[:len(prefix)] != prefix {
		return Digest{}, fmt.Errorf("digest %q must be of the form sha256:<64 hex>", s)
	}
	hex := s[len(prefix):]
	if len(hex) != sha256.Size*2 {
		return Digest{}, fmt.Errorf("digest %q: sha256 hex must be %d chars, got %d", s, sha256.Size*2, len(hex))
	}
	for i := 0; i < len(hex); i++ {
		c := hex[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return Digest{}, fmt.Errorf("digest %q: hex must be lowercase 0-9a-f", s)
		}
	}
	return Digest{Algo: "sha256", Hex: hex}, nil
}

// String renders the canonical "sha256:<hex>" form.
func (d Digest) String() string { return d.Algo + ":" + d.Hex }

// Equal reports whether two digests pin the same content.
func (d Digest) Equal(o Digest) bool { return d.Algo == o.Algo && d.Hex == o.Hex }

// IsZero reports whether the digest is unset.
func (d Digest) IsZero() bool { return d.Algo == "" && d.Hex == "" }
