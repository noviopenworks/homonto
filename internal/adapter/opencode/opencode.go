package opencode

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/noviopenworks/homonto/internal/adapter"
	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/fsutil"
	"github.com/noviopenworks/homonto/internal/jsonutil"
	"github.com/noviopenworks/homonto/internal/link"
	"github.com/noviopenworks/homonto/internal/secret"
	"github.com/noviopenworks/homonto/internal/skillpath"
	"github.com/noviopenworks/homonto/internal/state"
)

// Adapter projects desired config into OpenCode's opencode.jsonc under home.
type Adapter struct {
	home        string
	content     string
	scope       string // "" or "user" → home layout; "project" → projectRoot layout
	projectRoot string // directory of homonto.toml; used only for project scope
	skills      []string
}

// New builds an OpenCode adapter at user scope. home is $HOME; content holds
// owned skills. Use WithScope to install skills under a project root.
func New(home, content string) *Adapter { return &Adapter{home: home, content: content} }

// WithScope sets the skill install scope and project root (the homonto.toml
// directory). It affects skill symlink placement only — MCP servers, settings,
// and plugins always project under home. Empty scope means user scope. Returns
// the adapter for chaining.
func (a *Adapter) WithScope(scope, projectRoot string) *Adapter {
	a.scope, a.projectRoot = scope, projectRoot
	return a
}

func (a *Adapter) Name() string { return "opencode" }

func (a *Adapter) cfgFile() string {
	return filepath.Join(a.home, ".config", "opencode", "opencode.jsonc")
}

// skillsDir is the directory owned-skill symlinks live in for the active scope.
func (a *Adapter) skillsDir() string {
	return skillpath.Dir("opencode", a.scope, a.home, a.projectRoot)
}

// inactiveSkillsDir is the other scope's skills directory — where a link may
// linger after a scope switch. It returns "" when there is nothing meaningful
// to relocate from: no project root is known, or the two scopes resolve to the
// same directory.
func (a *Adapter) inactiveSkillsDir() string {
	if a.projectRoot == "" {
		return ""
	}
	d := skillpath.Dir("opencode", skillpath.Other(a.scope), a.home, a.projectRoot)
	if d == a.skillsDir() {
		return ""
	}
	return d
}

func (a *Adapter) desiredMCPs(c *config.Config) map[string]string {
	out := map[string]string{}
	for name, m := range c.MCPs {
		if !contains(m.TargetsOrAll(), "opencode") {
			continue
		}
		// No command means nothing runnable to project (matches claude's
		// adapter); writing `command: []` would just break the tool.
		if len(m.Command) == 0 {
			continue
		}
		obj := map[string]any{"type": "local", "command": m.Command, "enabled": true}
		if len(m.Env) > 0 {
			obj["environment"] = m.Env
		}
		out["mcp."+name] = mustJSON(obj)
	}
	return out
}

func (a *Adapter) Plan(c *config.Config, st *state.State) (adapter.ChangeSet, error) {
	a.skills = c.Skills.Own
	doc, err := readStandardized(a.cfgFile())
	if err != nil {
		return adapter.ChangeSet{}, err
	}
	cs := adapter.ChangeSet{Tool: "opencode"}

	// Config-supplied names are escaped so reads (and Apply's writes, which
	// escape the same way) address the literal key, not gjson path syntax.
	des := a.desiredMCPs(c)
	for key, want := range des {
		disk, hasDisk := jsonutil.GetJSON(doc, "mcp."+jsonutil.EscapePath(trim(key, "mcp.")))
		cs.Changes = append(cs.Changes, planKey(st, key, want, disk, hasDisk))
	}
	for k, v := range c.Settings.OpenCode {
		key := "setting." + k
		want := mustJSON(v)
		disk, hasDisk := jsonutil.GetJSON(doc, jsonutil.EscapePath(k))
		cs.Changes = append(cs.Changes, planKey(st, key, want, disk, hasDisk))
	}
	for _, p := range c.Plugins.OpenCode {
		if arrayHas(doc, "plugin", p) {
			// Present on disk. If recorded, steady-state noop; otherwise adopt it
			// into state so pruning and drift can see it (plugin names are plain,
			// never secret-bearing).
			if _, inState := st.Get("opencode", "plugin."+p); inState {
				cs.Changes = append(cs.Changes, adapter.Change{Action: "noop", Key: "plugin." + p})
			} else {
				cs.Changes = append(cs.Changes, adapter.Change{Action: "adopt", Key: "plugin." + p, New: mustJSON(p)})
			}
		} else {
			cs.Changes = append(cs.Changes, adapter.Change{Action: "create", Key: "plugin." + p, New: mustJSON(p)})
		}
	}
	ops, err := link.Plan(a.links())
	if err != nil {
		return adapter.ChangeSet{}, err
	}
	inactive := a.inactiveSkillsDir()
	for _, op := range ops {
		name := filepath.Base(op.Dst)
		// A create whose same-named link still exists (as our managed symlink) at
		// the other scope is a scope switch: render it as a relocate so the move —
		// and the prune of the old link Apply performs — is visible before confirm.
		if op.Cur == "" && inactive != "" && link.IsManaged(filepath.Join(inactive, name), a.content) {
			cs.Changes = append(cs.Changes, adapter.Change{Action: "update", Key: "skill." + name, Old: filepath.Join(inactive, name), New: op.Dst + " -> " + op.Src})
		} else if op.Cur == "" {
			cs.Changes = append(cs.Changes, adapter.Change{Action: "create", Key: "skill." + name, New: op.Dst + " -> " + op.Src})
		} else {
			cs.Changes = append(cs.Changes, adapter.Change{Action: "update", Key: "skill." + name, Old: op.Cur, New: op.Src})
		}
	}
	// Orphans: a state key no longer declared in config is de-declared — plan a
	// delete. (A declared key missing from disk is drift, handled above.) Old is
	// always redacted: a removed key's provenance is stale by definition.
	declared := map[string]bool{}
	for k := range des {
		declared[k] = true
	}
	for k := range c.Settings.OpenCode {
		declared["setting."+k] = true
	}
	for _, p := range c.Plugins.OpenCode {
		declared["plugin."+p] = true
	}
	for _, n := range c.Skills.Own {
		declared["skill."+n] = true
	}
	for _, k := range st.Keys("opencode") {
		if declared[k] || !managedPrefix(k) {
			continue
		}
		cs.Changes = append(cs.Changes, adapter.Change{Action: "delete", Key: k, Old: adapter.SecretRedaction})
	}
	// Keys come from map iteration (random order); a plan must render the
	// same way every run. Keys are unique within a changeset.
	sort.SliceStable(cs.Changes, func(i, j int) bool { return cs.Changes[i].Key < cs.Changes[j].Key })
	return cs, nil
}

// links maps each owned skill's destination to its content source.
func (a *Adapter) links() map[string]string {
	out := map[string]string{}
	for _, name := range a.skills {
		out[filepath.Join(a.skillsDir(), name)] = filepath.Join(a.content, "skills", name)
	}
	return out
}

// planKey applies the shared O-3 decision: direct compare for non-secret keys,
// token+hash compare for secret keys (never exposing the on-disk value).
func planKey(st *state.State, key, want, disk string, hasDisk bool) adapter.Change {
	e, inState := st.Get("opencode", key)
	switch {
	case !hasDisk:
		return adapter.Change{Action: "create", Key: key, New: want}
	case !secret.ContainsRef(want):
		if jsonutil.Canonical(disk) == jsonutil.Canonical(want) {
			// Disk already matches desired. A true noop requires state to also
			// already record this exact on-disk value; otherwise adopt it so
			// pruning and drift can see it and the stale/absent Applied hash is
			// refreshed (mirrors the secret branch below; secret keys never reach
			// this branch).
			if inState && e.Applied == secret.Hash(jsonutil.Canonical(disk)) {
				return adapter.Change{Action: "noop", Key: key}
			}
			return adapter.Change{Action: "adopt", Key: key, New: want}
		}
		old := disk
		// Never print the on-disk value when it may be a resolved secret: either
		// the key was previously a secret, or it is not in state at all (unknown
		// provenance — a lost state.json must not cause leaks).
		if !inState || secret.ContainsRef(e.Desired) {
			old = adapter.SecretRedaction
		}
		return adapter.Change{Action: "update", Key: key, Old: old, New: want}
	default:
		if inState && e.Desired == want && e.Applied == secret.Hash(jsonutil.Canonical(disk)) {
			return adapter.Change{Action: "noop", Key: key}
		}
		return adapter.Change{Action: "update", Key: key, Old: adapter.SecretRedaction, New: want}
	}
}

// ObserveHashes hashes the current on-disk value of every recorded key still
// present, so an unchanged key reproduces its Entry.Applied (mirroring claude,
// as far as opencode's data model allows). Only hashes escape — raw values
// (possibly resolved secrets) never leave the adapter.
func (a *Adapter) ObserveHashes(st *state.State) (map[string]string, error) {
	doc, err := readStandardized(a.cfgFile())
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, key := range st.Keys("opencode") {
		switch {
		case hasPrefix(key, "mcp."):
			if v, ok := jsonutil.GetJSON(doc, "mcp."+jsonutil.EscapePath(trim(key, "mcp."))); ok {
				out[key] = secret.Hash(jsonutil.Canonical(v))
			}
		case hasPrefix(key, "setting."):
			if v, ok := jsonutil.GetJSON(doc, jsonutil.EscapePath(trim(key, "setting."))); ok {
				out[key] = secret.Hash(jsonutil.Canonical(v))
			}
		case hasPrefix(key, "plugin."):
			// Plugins are array membership with no scalar to re-hash: presence
			// means unchanged by definition, so return the key's own Applied.
			if arrayHas(doc, "plugin", trim(key, "plugin.")) {
				if e, ok := st.Get("opencode", key); ok {
					out[key] = e.Applied
				}
			}
		case hasPrefix(key, "skill."):
			// skill.* is a symlink; its Applied was Hash(dst + " -> " + src).
			dst := filepath.Join(a.skillsDir(), trim(key, "skill."))
			target, err := os.Readlink(dst)
			if err != nil {
				continue // missing or not a symlink → omit
			}
			out[key] = secret.Hash(dst + " -> " + target)
		}
		// absent from disk → omit
	}
	return out, nil
}

func (a *Adapter) Apply(cs adapter.ChangeSet, res *secret.Resolver, st *state.State) error {
	doc, err := readStandardized(a.cfgFile())
	if err != nil {
		return err
	}
	// Write opencode.jsonc only when a managed key in it actually changed.
	// adopt/noop are state-only and must leave the file byte-for-byte untouched
	// (JSONC comments preserved); skill.* is symlink work, not JSON.
	docChanged := false
	for _, c := range cs.Changes {
		if c.Action == "noop" {
			continue
		}
		if c.Action == "adopt" {
			// Adoption records a pre-existing matching key into state without
			// touching the tool file. The on-disk value already equals want, so
			// the recorded Applied hash equals the hash of the on-disk value.
			val, err := res.ResolveJSON(c.New)
			if err != nil {
				return err
			}
			st.Set("opencode", c.Key, c.New, secret.Hash(jsonutil.Canonical(mustJSON(val))))
			continue
		}
		if c.Action == "delete" {
			switch {
			case hasPrefix(c.Key, "mcp."):
				doc, err = jsonutil.DeleteJSON(doc, "mcp."+jsonutil.EscapePath(trim(c.Key, "mcp.")))
				docChanged = true
			case hasPrefix(c.Key, "setting."):
				doc, err = jsonutil.DeleteJSON(doc, jsonutil.EscapePath(trim(c.Key, "setting.")))
				docChanged = true
			case hasPrefix(c.Key, "plugin."):
				doc, err = jsonutil.RemoveArrayElem(doc, "plugin", trim(c.Key, "plugin."))
				docChanged = true
			case hasPrefix(c.Key, "skill."):
				// Only a symlink into our content dir is removed; anything else
				// is a conflict error inside link.Remove.
				err = link.Remove(filepath.Join(a.skillsDir(), trim(c.Key, "skill.")), a.content)
			}
			if err != nil {
				return err
			}
			st.Delete("opencode", c.Key)
			continue
		}
		// skill.* changes are symlink work, handled below — not JSON keys.
		if hasPrefix(c.Key, "skill.") {
			continue
		}
		val, err := res.ResolveJSON(c.New)
		if err != nil {
			return err
		}
		// Escaped like Plan's reads: the write must land on the literal key.
		switch {
		case hasPrefix(c.Key, "mcp."):
			doc, err = jsonutil.SetJSON(doc, "mcp."+jsonutil.EscapePath(trim(c.Key, "mcp.")), val)
			docChanged = true
		case hasPrefix(c.Key, "setting."):
			doc, err = jsonutil.SetJSON(doc, jsonutil.EscapePath(trim(c.Key, "setting.")), val)
			docChanged = true
		case hasPrefix(c.Key, "plugin."):
			doc, err = jsonutil.EnsureArrayElem(doc, "plugin", trim(c.Key, "plugin."))
			docChanged = true
		}
		if err != nil {
			return err
		}
		st.Set("opencode", c.Key, c.New, secret.Hash(jsonutil.Canonical(mustJSON(val))))
	}
	// Fail fast on link conflicts before writing any file.
	links := a.links()
	if _, err := link.Plan(links); err != nil {
		return err
	}
	if docChanged {
		if err := fsutil.WriteAtomic(a.cfgFile(), doc); err != nil {
			return err
		}
	}
	// Prune a link left at the other scope after a scope switch, so no orphan
	// remains. Only our own managed symlink is removed (IsManaged guards it); a
	// foreign file or an absent path is left untouched — never an error.
	if inactive := a.inactiveSkillsDir(); inactive != "" {
		for _, name := range a.skills {
			old := filepath.Join(inactive, name)
			if link.IsManaged(old, a.content) {
				if err := link.Remove(old, a.content); err != nil {
					return err
				}
			}
		}
	}
	for dst, src := range links {
		if _, err := link.Link(src, dst); err != nil {
			return err
		}
		// Record the link in state so pruning sees de-declared skills later.
		st.Set("opencode", "skill."+filepath.Base(dst), dst+" -> "+src, secret.Hash(dst+" -> "+src))
	}
	return nil
}
