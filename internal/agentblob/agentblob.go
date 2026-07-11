// Package agentblob is a content-addressed store for installed agent base
// content (.homonto/agents-blobs/<sha256hex>). The blob key is exactly the
// lockfile install hash (agentlock.HashContent), so a future three-way update
// can retrieve the base an install was materialized from by its recorded hash.
package agentblob

import (
	"os"
	"path/filepath"
	"sort"

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

// Reclaim removes stored base blobs whose hash is not in referenced — the set of
// content hashes any lockfile install still points at. It is safe: content is
// addressed by hash, so an unreferenced blob can never be needed again (the only
// blob a future three-way update reads is the current recorded base,
// Install.Hash). With dryRun it removes nothing and only reports what would go.
// Returns the affected hashes, sorted. A missing store is empty (no error).
func Reclaim(homontoDir string, referenced map[string]bool, dryRun bool) ([]string, error) {
	d := dir(homontoDir)
	ents, err := os.ReadDir(d)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var dead []string
	for _, e := range ents {
		if e.IsDir() || referenced[e.Name()] {
			continue
		}
		dead = append(dead, e.Name())
	}
	sort.Strings(dead)
	if dryRun {
		return dead, nil
	}
	for _, h := range dead {
		if err := os.Remove(filepath.Join(d, h)); err != nil {
			return nil, err
		}
	}
	return dead, nil
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
