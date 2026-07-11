// Package agentblob is a content-addressed store for installed agent base
// content (.homonto/agents-blobs/<sha256hex>). The blob key is exactly the
// lockfile install hash (agentlock.HashContent), so a future three-way update
// can retrieve the base an install was materialized from by its recorded hash.
package agentblob

import (
	"os"
	"path/filepath"

	"github.com/noviopenworks/homonto/internal/agentlock"
	"github.com/noviopenworks/homonto/internal/fsutil"
)

// dir is the blob store directory under homontoDir.
func dir(homontoDir string) string { return filepath.Join(homontoDir, "agents-blobs") }

// Put stores content under its sha256 hex (matching the lockfile install hash)
// and returns that hash. It is idempotent: an existing blob is left untouched.
func Put(homontoDir string, content []byte) (hash string, err error) {
	hash = agentlock.HashContent(content)
	p := filepath.Join(dir(homontoDir), hash)
	if _, err := os.Stat(p); err == nil {
		return hash, nil // already stored — content-addressed, so identical.
	}
	if err := fsutil.WriteAtomic(p, content); err != nil {
		return "", err
	}
	return hash, nil
}

// Get reads the blob for hash. A missing blob returns (nil, false, nil); any
// other read error returns (nil, false, err).
func Get(homontoDir, hash string) (content []byte, ok bool, err error) {
	b, err := os.ReadFile(filepath.Join(dir(homontoDir), hash))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return b, true, nil
}
