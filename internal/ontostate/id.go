package ontostate

import (
	"crypto/rand"
	"encoding/hex"
)

// NewID returns a stable, name-independent change identifier: 8 hex characters
// from crypto/rand. Assigned once at `onto new` and never rewritten, so a
// change's identity survives a rename. crypto/rand keeps ids unique without a
// counter or a wall clock.
func NewID() string {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand.Read never returns an error on supported platforms; if it
		// somehow does, an empty id is safer than a panic (the change still works,
		// just without a stable id — surfaced as empty like a legacy state).
		return ""
	}
	return hex.EncodeToString(b[:])
}
