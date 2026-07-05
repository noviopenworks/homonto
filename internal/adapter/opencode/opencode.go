package opencode

import (
	"path/filepath"
	"sort"

	"github.com/noviopenworks/homonto/internal/adapter"
	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/fsutil"
	"github.com/noviopenworks/homonto/internal/jsonutil"
	"github.com/noviopenworks/homonto/internal/link"
	"github.com/noviopenworks/homonto/internal/secret"
	"github.com/noviopenworks/homonto/internal/state"
)

// Adapter projects desired config into OpenCode's opencode.jsonc under home.
type Adapter struct {
	home    string
	content string
	skills  []string
}

// New builds an OpenCode adapter. home is $HOME; content holds owned skills.
func New(home, content string) *Adapter { return &Adapter{home: home, content: content} }

func (a *Adapter) Name() string { return "opencode" }

func (a *Adapter) cfgFile() string {
	return filepath.Join(a.home, ".config", "opencode", "opencode.jsonc")
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
	for _, op := range ops {
		if op.Cur == "" {
			cs.Changes = append(cs.Changes, adapter.Change{Action: "create", Key: "skill." + filepath.Base(op.Dst), New: op.Dst + " -> " + op.Src})
		} else {
			cs.Changes = append(cs.Changes, adapter.Change{Action: "update", Key: "skill." + filepath.Base(op.Dst), Old: op.Cur, New: op.Src})
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
		out[filepath.Join(a.home, ".config", "opencode", "skills", name)] = filepath.Join(a.content, "skills", name)
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
			// Disk already matches desired. If the key is recorded, this is a
			// steady-state noop; otherwise adopt it into state so pruning and
			// drift can see it (secret keys never reach this branch).
			if inState {
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

func (a *Adapter) Apply(cs adapter.ChangeSet, res *secret.Resolver, st *state.State) error {
	doc, err := readStandardized(a.cfgFile())
	if err != nil {
		return err
	}
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
			case hasPrefix(c.Key, "setting."):
				doc, err = jsonutil.DeleteJSON(doc, jsonutil.EscapePath(trim(c.Key, "setting.")))
			case hasPrefix(c.Key, "plugin."):
				doc, err = jsonutil.RemoveArrayElem(doc, "plugin", trim(c.Key, "plugin."))
			case hasPrefix(c.Key, "skill."):
				// Only a symlink into our content dir is removed; anything else
				// is a conflict error inside link.Remove.
				err = link.Remove(filepath.Join(a.home, ".config", "opencode", "skills", trim(c.Key, "skill.")), a.content)
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
		case hasPrefix(c.Key, "setting."):
			doc, err = jsonutil.SetJSON(doc, jsonutil.EscapePath(trim(c.Key, "setting.")), val)
		case hasPrefix(c.Key, "plugin."):
			doc, err = jsonutil.EnsureArrayElem(doc, "plugin", trim(c.Key, "plugin."))
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
	if err := fsutil.WriteAtomic(a.cfgFile(), doc); err != nil {
		return err
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
