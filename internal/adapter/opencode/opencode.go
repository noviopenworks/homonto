package opencode

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

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
	catalogRoot string // materialized builtin catalog root (.homonto/catalog/skills)
	projectRoot string // directory of homonto.toml; used for project-scope resources
	skills      []config.NamedResource
}

// New builds an OpenCode adapter at user scope. home is $HOME; content holds
// owned skills. Use WithProjectRoot to install project-scope skills.
func New(home, content string) *Adapter { return &Adapter{home: home, content: content} }

// WithProjectRoot sets the project root (the homonto.toml directory). It is
// used for project-scope resource placement. MCP servers, settings, and
// plugins always project under home.
func (a *Adapter) WithProjectRoot(projectRoot string) *Adapter {
	a.projectRoot = projectRoot
	return a
}

// WithCatalogRoot sets the materialized builtin-catalog root that builtin:<name>
// skills link from. Mirrors WithProjectRoot.
func (a *Adapter) WithCatalogRoot(catalogRoot string) *Adapter {
	a.catalogRoot = catalogRoot
	return a
}

// managedRoots returns every content root homonto owns links into. catalogRoot
// is included only when set: link.managed() treats an empty-string root as a
// prefix match for every absolute path, so passing "" here would make link
// calls treat any symlink as "ours" — an empty catalogRoot must never reach
// link.*.
func (a *Adapter) managedRoots() []string {
	roots := []string{a.content}
	if a.catalogRoot != "" {
		roots = append(roots, a.catalogRoot)
	}
	return roots
}

// skillSource resolves a skill entry's on-disk content directory by source
// scheme: builtin:<n> from the materialized catalog root, otherwise the local
// content dir.
func (a *Adapter) skillSource(entry config.NamedResource) string {
	if s := entry.Resource.Source; strings.HasPrefix(s, "builtin:") {
		return filepath.Join(a.catalogRoot, strings.TrimPrefix(s, "builtin:"))
	}
	return filepath.Join(a.content, "skills", localSourceName(entry.Resource.Source, entry.Name))
}

func (a *Adapter) Name() string { return "opencode" }

func (a *Adapter) cfgFile() string {
	return filepath.Join(a.home, ".config", "opencode", "opencode.jsonc")
}

// skillsDir is the directory owned-skill symlinks live in for the given scope.
func (a *Adapter) skillsDir(scope string) string {
	return skillpath.Dir("opencode", scope, a.home, a.projectRoot)
}

// inactiveSkillsDir is the other scope's skills directory — where a link may
// linger after a per-resource scope switch. It returns "" when there is nothing
// meaningful to relocate from: no project root is known, or the two scopes
// resolve to the same directory.
func (a *Adapter) inactiveSkillsDir(scope string) string {
	if a.projectRoot == "" {
		return ""
	}
	d := skillpath.Dir("opencode", skillpath.Other(scope), a.home, a.projectRoot)
	if d == a.skillsDir(scope) {
		return ""
	}
	return d
}

// localSourceName resolves a skill resource's content subdirectory: a local:
// source names that directory directly; any other source falls back to the
// skill's declared name.
func localSourceName(source, fallback string) string {
	if strings.HasPrefix(source, "local:") {
		return strings.TrimPrefix(source, "local:")
	}
	return fallback
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
	skills, err := c.ExpandedSkillEntriesForTool("opencode")
	if err != nil {
		return adapter.ChangeSet{}, err
	}
	a.skills = skills
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
	ops, err := link.Plan(a.links(), a.managedRoots()...)
	if err != nil {
		return adapter.ChangeSet{}, err
	}
	entryByName := map[string]config.NamedResource{}
	for _, entry := range a.skills {
		entryByName[entry.Name] = entry
	}
	for _, op := range ops {
		name := filepath.Base(op.Dst)
		entry := entryByName[name]
		inactive := a.inactiveSkillsDir(entry.Resource.Scope)
		// A create whose same-named link still exists (as our managed symlink) at
		// the other scope is a scope switch: render it as a relocate so the move —
		// and the prune of the old link Apply performs — is visible before confirm.
		if op.Cur == "" && inactive != "" && link.IsManaged(filepath.Join(inactive, name), a.managedRoots()...) {
			cs.Changes = append(cs.Changes, adapter.Change{Action: "update", Key: "skill." + name, Old: filepath.Join(inactive, name), New: op.Dst + " -> " + op.Src})
		} else if op.Cur == "" {
			cs.Changes = append(cs.Changes, adapter.Change{Action: "create", Key: "skill." + name, New: op.Dst + " -> " + op.Src})
		} else {
			cs.Changes = append(cs.Changes, adapter.Change{Action: "update", Key: "skill." + name, Old: op.Cur, New: op.Src})
		}
	}
	// Adopt a correct-but-unrecorded skill link — one already on disk pointing at
	// its content, but absent from state (or stale). link.Plan omits a correct
	// link, so without this a lost state.json for a skills-only config could never
	// be rebuilt (apply short-circuits with no change). Mirrors mcp/setting/plugin
	// adoption: state-only, the on-disk link is left untouched.
	opDst := map[string]bool{}
	for _, op := range ops {
		opDst[op.Dst] = true
	}
	for _, entry := range a.skills {
		name := entry.Name
		dst := filepath.Join(a.skillsDir(entry.Resource.Scope), name)
		if opDst[dst] {
			continue // a create/relink/relocate already covers it
		}
		src := a.skillSource(entry)
		if tgt, err := os.Readlink(dst); err != nil || tgt != src {
			continue // not a correct link into content
		}
		if e, ok := st.Get("opencode", "skill."+name); ok && e.Applied == secret.Hash(dst+" -> "+src) {
			continue // already recorded → a true noop, nothing to do
		}
		cs.Changes = append(cs.Changes, adapter.Change{Action: "adopt", Key: "skill." + name, New: dst + " -> " + src})
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
	for _, entry := range a.skills {
		declared["skill."+entry.Name] = true
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

// links maps each owned skill's destination to its content source. Each skill
// resource carries its own scope, so dst is computed per entry.
func (a *Adapter) links() map[string]string {
	out := map[string]string{}
	for _, entry := range a.skills {
		out[filepath.Join(a.skillsDir(entry.Resource.Scope), entry.Name)] = a.skillSource(entry)
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
			// Reproduce it by reading the link at the dst state recorded — NOT the
			// current scope's skillsDir. A pending [skills] scope switch changes
			// skillsDir but leaves the applied link in place; reading the new scope's
			// (empty) location would make an intact old link look "missing" (false
			// drift) instead of the pending relocation Plan already surfaces.
			e, ok := st.Get("opencode", key)
			if !ok {
				continue
			}
			dst, ok := recordedDst(e.Desired)
			if !ok {
				continue
			}
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
			// A skill adoption records a correct-but-unrecorded symlink into state
			// without touching disk; its value is "dst -> src", not JSON, so it is
			// recorded exactly like a freshly linked skill (Hash of "dst -> src").
			if hasPrefix(c.Key, "skill.") {
				st.Set("opencode", c.Key, c.New, secret.Hash(c.New))
				continue
			}
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
				// is a conflict error inside link.Remove. A de-declared skill is
				// no longer in a.skills, so recover the on-disk location from the
				// dst state recorded for it (per-resource scope: each skill lives
				// at exactly one place). Fall back to user scope when state is
				// missing the recorded dst.
				name := trim(c.Key, "skill.")
				dst := ""
				if e, ok := st.Get("opencode", c.Key); ok {
					dst, _ = recordedDst(e.Desired)
				}
				if dst == "" {
					dst = filepath.Join(a.skillsDir("user"), name)
				}
				err = link.Remove(dst, a.managedRoots()...)
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
	if _, err := link.Plan(links, a.managedRoots()...); err != nil {
		return err
	}
	if docChanged {
		if err := fsutil.WriteAtomic(a.cfgFile(), doc); err != nil {
			return err
		}
	}
	// Prune a link left at a skill's inactive scope after a per-resource scope
	// switch, so no orphan remains. Only our own managed symlink is removed
	// (IsManaged guards it); a foreign file or an absent path is left untouched
	// — never an error. Each skill carries its own scope, so the inactive dir is
	// computed per entry.
	for _, entry := range a.skills {
		inactive := a.inactiveSkillsDir(entry.Resource.Scope)
		if inactive == "" {
			continue
		}
		old := filepath.Join(inactive, entry.Name)
		if link.IsManaged(old, a.managedRoots()...) {
			if err := link.Remove(old, a.managedRoots()...); err != nil {
				return err
			}
		}
	}
	for dst, src := range links {
		if _, err := link.Link(src, dst, a.managedRoots()...); err != nil {
			return err
		}
		// Record the link in state so pruning sees de-declared skills later.
		st.Set("opencode", "skill."+filepath.Base(dst), dst+" -> "+src, secret.Hash(dst+" -> "+src))
	}
	return nil
}
