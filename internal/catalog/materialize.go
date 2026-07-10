package catalog

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Materialize extracts each named builtin skill from the embedded FS into
// dstRoot/<name>/, removing any existing per-skill directory first so a stale
// file from a previous version cannot survive an upgrade. It is the caller's
// job (engine) to gate this on the catalog version.
func (c *Catalog) Materialize(dstRoot string, skillNames []string) error {
	for _, name := range skillNames {
		sp, ok := c.skills[name]
		if !ok {
			return fmt.Errorf("catalog: unknown skill %q", name)
		}
		sub, err := fs.Sub(c.fsys, sp)
		if err != nil {
			return fmt.Errorf("catalog: sub %q: %w", sp, err)
		}
		dstDir := filepath.Join(dstRoot, name)
		if err := os.RemoveAll(dstDir); err != nil {
			return err
		}
		err = fs.WalkDir(sub, ".", func(p string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			target := filepath.Join(dstDir, filepath.FromSlash(p))
			if d.IsDir() {
				return os.MkdirAll(target, 0o755)
			}
			data, err := fs.ReadFile(sub, p)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			return os.WriteFile(target, data, 0o644)
		})
		if err != nil {
			return err
		}
	}
	return nil
}
