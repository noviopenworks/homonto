package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/fsutil"
	"github.com/noviopenworks/homonto/internal/remote"
)

// remotePaths returns the lockfile and revocation-list paths under the state dir.
func (e *Engine) remoteLockPath() string { return filepath.Join(e.StateDir, "remote.lock.json") }
func (e *Engine) remoteRevokedPath() string {
	return filepath.Join(e.StateDir, "revoked.json")
}

// materializeRemotes resolves every declared remote subagent through the trust
// pipeline (fetch → validate → pin-match → revocation → cache), materializes the
// verified content into the deterministic remote root, records provenance in the
// remote lockfile, prunes de-declared installs, and GCs unreferenced cache
// entries. It aborts before any adapter write if a remote resource fails closed.
func (e *Engine) materializeRemotes() error {
	declared, err := e.declaredRemoteSubagents()
	if err != nil {
		return err
	}

	rev, err := remote.LoadRevocations(e.remoteRevokedPath())
	if err != nil {
		return err
	}
	resolver := &remote.Resolver{
		Cache:       &remote.Cache{Root: e.RemoteCacheRoot},
		Revocations: rev,
		Limits:      remote.DefaultLimits,
	}
	lock, err := remote.LoadLock(e.remoteLockPath())
	if err != nil {
		return err
	}

	subagentRoot := filepath.Join(e.RemoteRoot, "subagents")

	// Prune remote-content files and lock entries for subagents no longer declared.
	if err := e.pruneRemoteSubagents(&lock, declared, subagentRoot); err != nil {
		return err
	}

	names := make([]string, 0, len(declared))
	for name := range declared {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		res := declared[name]
		src, err := remote.ParseRemoteSource(res.Source)
		if err != nil {
			return fmt.Errorf("remote subagent %q: %w", name, err)
		}
		pin, err := remote.ParseDigest(res.Digest)
		if err != nil {
			return fmt.Errorf("remote subagent %q: %w", name, err)
		}
		cacheDir, err := resolver.Resolve(context.Background(), src, pin)
		if err != nil {
			return fmt.Errorf("remote subagent %q: %w", name, err)
		}
		if err := materializeRemoteFile(cacheDir, name, subagentRoot); err != nil {
			return fmt.Errorf("remote subagent %q: %w", name, err)
		}
		lock.Set(remote.LockEntry{
			Kind:      "subagent",
			Name:      name,
			Locator:   src.URL,
			Transport: string(src.Transport),
			Digest:    pin.String(),
			Size:      fileSize(filepath.Join(subagentRoot, name+".md")),
		})
	}

	if err := lock.Save(e.remoteLockPath()); err != nil {
		return err
	}
	// Reclaim cache entries no lock entry references.
	if _, err := resolver.Cache.GC(lock.Digests(), false); err != nil {
		return err
	}
	return nil
}

// declaredRemoteSubagents collects the remote subagents declared for either
// tool, de-duplicated by name (a subagent targeting both tools appears once).
func (e *Engine) declaredRemoteSubagents() (map[string]config.Resource, error) {
	out := map[string]config.Resource{}
	for _, tool := range []string{"claude", "opencode"} {
		entries, err := e.Cfg.ExpandedSubagentEntriesForTool(tool)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if remote.IsRemoteSource(entry.Resource.Source) {
				out[entry.Name] = entry.Resource
			}
		}
	}
	return out, nil
}

// pruneRemoteSubagents removes materialized content files and lock entries for
// remote subagents that are recorded but no longer declared.
func (e *Engine) pruneRemoteSubagents(lock *remote.Lock, declared map[string]config.Resource, subagentRoot string) error {
	for key, entry := range lock.Entries {
		if entry.Kind != "subagent" {
			continue
		}
		if _, ok := declared[entry.Name]; ok {
			continue
		}
		if err := os.Remove(filepath.Join(subagentRoot, entry.Name+".md")); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remote: pruning %q: %w", entry.Name, err)
		}
		delete(lock.Entries, key)
	}
	return nil
}

// materializeRemoteFile copies <cacheDir>/<name>.md into the remote content root
// atomically. A remote subagent archive must contain <name>.md at its root.
func materializeRemoteFile(cacheDir, name, destRoot string) error {
	srcFile := filepath.Join(cacheDir, name+".md")
	data, err := os.ReadFile(srcFile)
	if err != nil {
		return fmt.Errorf("remote archive must contain %q at its root: %w", name+".md", err)
	}
	if err := os.MkdirAll(destRoot, 0o755); err != nil {
		return err
	}
	return fsutil.WriteAtomic(filepath.Join(destRoot, name+".md"), data)
}

func fileSize(p string) int64 {
	if info, err := os.Stat(p); err == nil {
		return info.Size()
	}
	return 0
}
