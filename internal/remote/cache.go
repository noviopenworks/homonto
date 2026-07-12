package remote

import (
	"fmt"
	"os"
	"path/filepath"
)

// Cache is a content-addressed store for verified remote trees. A tree keyed by
// its canonical digest resolves offline and reproducibly.
type Cache struct {
	Root string // e.g. .homonto/cache/remote
}

// algoDir/Dir lay out <Root>/<algo>/<hex>/ so different algorithms never alias.
func (c *Cache) Dir(d Digest) string {
	return filepath.Join(c.Root, d.Algo, d.Hex)
}

// Has reports whether the digest's content is already materialized.
func (c *Cache) Has(d Digest) bool {
	info, err := os.Stat(c.Dir(d))
	return err == nil && info.IsDir()
}

// Put materializes a tree at its digest's cache directory atomically (staging
// dir + rename). A second Put of the same digest is a no-op returning the
// existing path. The caller is responsible for having verified the tree's
// digest before Put.
func (c *Cache) Put(d Digest, tree Tree) (string, error) {
	dest := c.Dir(d)
	if c.Has(d) {
		return dest, nil
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", fmt.Errorf("remote: cache: %w", err)
	}
	staging, err := os.MkdirTemp(filepath.Dir(dest), ".staging-*")
	if err != nil {
		return "", fmt.Errorf("remote: cache: %w", err)
	}
	// On any error before the final rename, remove the staging dir.
	committed := false
	defer func() {
		if !committed {
			os.RemoveAll(staging)
		}
	}()
	if err := writeTree(staging, tree); err != nil {
		return "", err
	}
	if err := os.Rename(staging, dest); err != nil {
		// A concurrent writer may have won the race; if the dest now exists, accept
		// it and let the deferred cleanup remove our stale staging dir.
		if c.Has(d) {
			return dest, nil
		}
		return "", fmt.Errorf("remote: cache commit: %w", err)
	}
	committed = true
	return dest, nil
}

// GC removes cache directories whose digest is not in referenced. With dryRun it
// reports what would be removed without deleting. It returns the reclaimed
// digests.
func (c *Cache) GC(referenced []Digest, dryRun bool) ([]Digest, error) {
	keep := map[string]bool{}
	for _, d := range referenced {
		keep[d.Algo+"/"+d.Hex] = true
	}
	var reclaimed []Digest
	algos, err := os.ReadDir(c.Root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("remote: cache gc: %w", err)
	}
	for _, algo := range algos {
		if !algo.IsDir() {
			continue
		}
		algoPath := filepath.Join(c.Root, algo.Name())
		entries, err := os.ReadDir(algoPath)
		if err != nil {
			return nil, fmt.Errorf("remote: cache gc: %w", err)
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			key := algo.Name() + "/" + e.Name()
			if keep[key] {
				continue
			}
			reclaimed = append(reclaimed, Digest{Algo: algo.Name(), Hex: e.Name()})
			if !dryRun {
				if err := os.RemoveAll(filepath.Join(algoPath, e.Name())); err != nil {
					return nil, fmt.Errorf("remote: cache gc: %w", err)
				}
			}
		}
	}
	return reclaimed, nil
}

// writeTree materializes a validated tree under dir, creating parent
// directories and preserving the executable bit.
func writeTree(dir string, tree Tree) error {
	for _, f := range tree.Files {
		dest := filepath.Join(dir, filepath.FromSlash(f.Path))
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return fmt.Errorf("remote: materialize: %w", err)
		}
		mode := os.FileMode(0o644)
		if f.Mode&0o111 != 0 {
			mode = 0o755
		}
		if err := os.WriteFile(dest, f.Data, mode); err != nil {
			return fmt.Errorf("remote: materialize: %w", err)
		}
	}
	return nil
}
