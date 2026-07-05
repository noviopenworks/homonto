// Package fsutil holds shared filesystem helpers used by adapters and state.
package fsutil

import (
	"os"
	"path/filepath"
)

// WriteAtomic writes data to path via a unique temp file in the target
// directory, fsyncing before rename so a crash never leaves a truncated
// file. An existing file keeps its current mode (a user-tightened 0600 is
// never loosened); new files default to 0600 because managed configs may
// receive resolved secrets.
func WriteAtomic(path string, data []byte) error {
	// A symlinked target (e.g. ~/.claude.json -> dotfiles/claude.json) must be
	// written through, not replaced: renaming over the link path would swap it
	// for a regular file that silently diverges from the dotfiles copy. Write
	// at the resolved location instead; a missing file resolves to path as-is.
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	mode := os.FileMode(0o600)
	if fi, err := os.Stat(path); err == nil {
		mode = fi.Mode().Perm()
	}
	f, err := os.CreateTemp(dir, ".homonto-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp) // no-op once renamed
	if err := f.Chmod(mode); err != nil {
		f.Close()
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
