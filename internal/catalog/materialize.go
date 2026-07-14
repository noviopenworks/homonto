package catalog

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/noviopenworks/homonto/internal/agentfm"
	"github.com/noviopenworks/homonto/internal/fsutil"
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
		sub, err := fs.Sub(c.skillFS[name], sp)
		if err != nil {
			return fmt.Errorf("catalog: sub %q: %w", sp, err)
		}
		dstDir := filepath.Join(dstRoot, name)
		// Stage-then-swap so a read error, full disk, or crash mid-walk never
		// leaves a partially-written skill dir (which allSkillDirsExist would
		// mistake for complete and never repair). Write into a sibling staging
		// dir, then atomically swap it into place only after the whole walk
		// succeeds. Discard any leftover staging from a prior crashed run first.
		staging := dstDir + ".staging"
		if err := os.RemoveAll(staging); err != nil {
			return err
		}
		err = fs.WalkDir(sub, ".", func(p string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			target := filepath.Join(staging, filepath.FromSlash(p))
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
			// Catalog files live under .homonto (control plane); write no-follow
			// so a planted symlink cannot redirect materialization.
			return fsutil.WriteControlPlane(target, data, 0o644)
		})
		if err != nil {
			// dstDir is untouched (still the prior complete version); drop the
			// partial staging so the next run starts clean.
			_ = os.RemoveAll(staging)
			return err
		}
		// Swap: remove the old dir, then rename staging into place. A crash in
		// this window leaves dstDir absent (not partial), so the next run
		// re-materializes rather than trusting a half-written directory.
		if err := os.RemoveAll(dstDir); err != nil {
			_ = os.RemoveAll(staging)
			return err
		}
		if err := os.Rename(staging, dstDir); err != nil {
			return err
		}
	}
	return nil
}

// MaterializeCommands writes each named builtin command from the embedded FS to
// dstRoot/<name>.md (a single file), replacing any existing file. Unlike
// Materialize (per-skill directories), no RemoveAll is needed — a single-file
// overwrite fully replaces prior content on upgrade. It is the caller's job
// (engine) to gate this on the catalog version.
func (c *Catalog) MaterializeCommands(dstRoot string, names []string) error {
	for _, name := range names {
		cp, ok := c.commands[name]
		if !ok {
			return fmt.Errorf("catalog: unknown command %q", name)
		}
		data, err := fs.ReadFile(c.commandFS[name], cp)
		if err != nil {
			return fmt.Errorf("catalog: read %q: %w", cp, err)
		}
		if err := os.MkdirAll(dstRoot, 0o755); err != nil {
			return err
		}
		if err := fsutil.WriteControlPlane(filepath.Join(dstRoot, name+".md"), data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// MaterializeSubagents writes each named builtin subagent from the embedded FS
// to dstRoot/<name>.md (a single file), replacing any existing file
// byte-for-byte. Like MaterializeCommands, no RemoveAll is needed — a
// single-file overwrite fully replaces prior content on upgrade. It is the
// caller's job (engine) to gate this on the catalog version.
//
// When a subagent's frontmatter carries a neutral `homonto:` access block (see
// internal/agentfm), it ALSO writes per-tool variants — <name>.claude.md and
// <name>.opencode.md — rendered from that block into each tool's native fields
// (Claude's `tools:` allowlist vs OpenCode's `permission:` map), which cannot
// share one file. renderCtx supplies the config-derived values the render needs
// (the role→model routes) per tool. A render that returns no bytes (a primary
// agent has no Claude variant) removes any stale variant instead of writing one,
// so the adapter's "block present + variant absent → skip" rule holds. Each
// adapter prefers its own variant; the shared <name>.md remains the version-gate
// anchor and the fallback for verbatim subagents.
func (c *Catalog) MaterializeSubagents(dstRoot string, names []string, renderCtx map[string]agentfm.RenderContext) error {
	for _, name := range names {
		sp, ok := c.subagents[name]
		if !ok {
			return fmt.Errorf("catalog: unknown subagent %q", name)
		}
		data, err := fs.ReadFile(c.subagentFS[name], sp)
		if err != nil {
			return fmt.Errorf("catalog: read %q: %w", sp, err)
		}
		if err := os.MkdirAll(dstRoot, 0o755); err != nil {
			return err
		}
		if err := fsutil.WriteControlPlane(filepath.Join(dstRoot, name+".md"), data, 0o644); err != nil {
			return err
		}
		if !agentfm.NeedsTransform(data) {
			continue
		}
		for _, tool := range []string{"claude", "opencode"} {
			rendered, rerr := agentfm.Render(data, tool, renderCtx[tool])
			if rerr != nil {
				return fmt.Errorf("catalog: render subagent %q for %s: %w", name, tool, rerr)
			}
			variant := filepath.Join(dstRoot, name+"."+tool+".md")
			if rendered == nil {
				if err := os.Remove(variant); err != nil && !os.IsNotExist(err) {
					return err
				}
				continue
			}
			if err := fsutil.WriteControlPlane(variant, rendered, 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}
