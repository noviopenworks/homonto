package secret

import (
	"crypto/sha256"
	"encoding/hex"
)

// Hash returns the lowercase hex sha256 of s. It is used to record a non-secret
// fingerprint of a resolved value in state, so idempotency and drift can be
// checked without storing or printing the plaintext.
func Hash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
