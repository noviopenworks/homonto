package claude

import (
	"encoding/json"
	"fmt"
	"os"
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

// Adapter projects desired config into Claude Code's files under home.
type Adapter struct {
	home    string
	content string
	skills  []string
}

// New builds a Claude adapter. home is the $HOME root; content holds owned
// skills.
func New(home, content string) *Adapter { return &Adapter{home: home, content: content} }

func (a *Adapter) Name() string { return "claude" }

func (a *Adapter) claudeJSON() string   { return filepath.Join(a.home, ".claude.json") }
func (a *Adapter) settingsJSON() string { return filepath.Join(a.home, ".claude", "settings.json") }

// links maps each owned skill's destination to its content source.
func (a *Adapter) links() map[string]string {
	out := map[string]string{}
	for _, name := range a.skills {
		out[filepath.Join(a.home, ".claude", "skills", name)] = filepath.Join(a.content, "skills", name)
	}
	return out
}

// desired returns managed key -> unresolved JSON-encoded desired value.
func (a *Adapter) desired(c *config.Config) map[string]string {
	out := map[string]string{}
	for name, m := range c.MCPs {
		if !contains(m.TargetsOrAll(), "claude") {
			continue
		}
		if len(m.Command) == 0 {
			continue
		}
		// Claude Code's real schema: command is a string with a separate args
		// array (matching `claude mcp add` output; empty keys omitted).
		obj := map[string]any{"type": "stdio", "command": m.Command[0]}
		if len(m.Command) > 1 {
			obj["args"] = m.Command[1:]
		}
		if len(m.Env) > 0 {
			obj["env"] = m.Env
		}
		out["mcp."+name] = mustJSON(obj)
	}
	for k, v := range c.Settings.Claude {
		out["setting."+k] = mustJSON(v)
	}
	for _, p := range c.Plugins.Claude {
		out["plugin."+p] = `true`
	}
	return out
}

func (a *Adapter) Plan(c *config.Config, st *state.State) (adapter.ChangeSet, error) {
	a.skills = c.Skills.Own
	cur, err := a.current()
	if err != nil {
		return adapter.ChangeSet{}, err
	}
	cs := adapter.ChangeSet{Tool: "claude"}
	des := a.desired(c)
	for key, want := range des {
		disk, hasDisk := cur[key]
		e, inState := st.Get("claude", key)
		switch {
		case !hasDisk:
			cs.Changes = append(cs.Changes, adapter.Change{Action: "create", Key: key, New: want})
		case !secret.ContainsRef(want):
			if jsonutil.Canonical(disk) == jsonutil.Canonical(want) {
				// Disk already matches desired. If the key is recorded, this is a
				// steady-state noop; otherwise adopt it into state so pruning and
				// drift can see it (secret keys never reach this branch).
				if inState {
					cs.Changes = append(cs.Changes, adapter.Change{Action: "noop", Key: key})
				} else {
					cs.Changes = append(cs.Changes, adapter.Change{Action: "adopt", Key: key, New: want})
				}
			} else {
				old := disk
				// Never print the on-disk value when it may be a resolved secret:
				// either the key was previously a secret, or it is not in state at
				// all (unknown provenance — a lost state.json must not cause leaks).
				if !inState || secret.ContainsRef(e.Desired) {
					old = adapter.SecretRedaction
				}
				cs.Changes = append(cs.Changes, adapter.Change{Action: "update", Key: key, Old: old, New: want})
			}
		default: // secret-bearing key: never expose the on-disk resolved value
			if inState && e.Desired == want && e.Applied == secret.Hash(jsonutil.Canonical(disk)) {
				cs.Changes = append(cs.Changes, adapter.Change{Action: "noop", Key: key})
			} else {
				cs.Changes = append(cs.Changes, adapter.Change{Action: "update", Key: key, Old: adapter.SecretRedaction, New: want})
			}
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
	for _, n := range c.Skills.Own {
		declared["skill."+n] = true
	}
	for _, k := range st.Keys("claude") {
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

// current reads existing managed values from disk, keyed like desired().
func (a *Adapter) current() (map[string]string, error) {
	out := map[string]string{}
	mj, err := readStandardized(a.claudeJSON())
	if err != nil {
		return nil, err
	}
	sj, err := readStandardized(a.settingsJSON())
	if err != nil {
		return nil, err
	}
	for k, v := range objMembers(mj, "mcpServers") {
		out["mcp."+k] = v
	}
	for k, v := range objMembers(sj, "enabledPlugins") {
		out["plugin."+k] = v
	}
	var m map[string]json.RawMessage
	_ = json.Unmarshal(sj, &m)
	for k, raw := range m {
		if k == "mcpServers" || k == "enabledPlugins" {
			continue
		}
		out["setting."+k] = string(raw)
	}
	return out, nil
}

func (a *Adapter) Apply(cs adapter.ChangeSet, res *secret.Resolver, st *state.State) error {
	mj, err := readStandardized(a.claudeJSON())
	if err != nil {
		return err
	}
	sj, err := readStandardized(a.settingsJSON())
	if err != nil {
		return err
	}
	// Write a tool file only when a managed key living in it actually changed.
	// adopt/noop are state-only and must leave the file byte-for-byte untouched
	// (comments/formatting preserved); skill.* is symlink work, not JSON.
	mjChanged, sjChanged := false, false
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
			st.Set("claude", c.Key, c.New, secret.Hash(jsonutil.Canonical(mustJSON(val))))
			continue
		}
		if c.Action == "delete" {
			switch {
			case hasPrefix(c.Key, "mcp."):
				mj, err = jsonutil.DeleteJSON(mj, "mcpServers."+jsonutil.EscapePath(trim(c.Key, "mcp.")))
				mjChanged = true
			case hasPrefix(c.Key, "setting."):
				sj, err = jsonutil.DeleteJSON(sj, jsonutil.EscapePath(trim(c.Key, "setting.")))
				sjChanged = true
			case hasPrefix(c.Key, "plugin."):
				sj, err = jsonutil.DeleteJSON(sj, "enabledPlugins."+jsonutil.EscapePath(trim(c.Key, "plugin.")))
				sjChanged = true
			case hasPrefix(c.Key, "skill."):
				// Only a symlink into our content dir is removed; anything else
				// is a conflict error inside link.Remove.
				err = link.Remove(filepath.Join(a.home, ".claude", "skills", trim(c.Key, "skill.")), a.content)
			}
			if err != nil {
				return err
			}
			st.Delete("claude", c.Key)
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
		// Config-supplied names are escaped so sjson writes the literal key
		// (matching current()'s literal reads) instead of nesting on dots or
		// silently dropping the write on @, | or #.
		switch {
		case hasPrefix(c.Key, "mcp."):
			mj, err = jsonutil.SetJSON(mj, "mcpServers."+jsonutil.EscapePath(trim(c.Key, "mcp.")), val)
			mjChanged = true
		case hasPrefix(c.Key, "setting."):
			sj, err = jsonutil.SetJSON(sj, jsonutil.EscapePath(trim(c.Key, "setting.")), val)
			sjChanged = true
		case hasPrefix(c.Key, "plugin."):
			sj, err = jsonutil.SetJSON(sj, "enabledPlugins."+jsonutil.EscapePath(trim(c.Key, "plugin.")), val)
			sjChanged = true
		}
		if err != nil {
			return err
		}
		// Store the unresolved form + a non-secret hash of the resolved value.
		st.Set("claude", c.Key, c.New, secret.Hash(jsonutil.Canonical(mustJSON(val))))
	}
	// Fail fast on link conflicts before writing any file.
	links := a.links()
	if _, err := link.Plan(links); err != nil {
		return err
	}
	if mjChanged {
		if err := fsutil.WriteAtomic(a.claudeJSON(), mj); err != nil {
			return err
		}
	}
	if sjChanged {
		if err := fsutil.WriteAtomic(a.settingsJSON(), sj); err != nil {
			return err
		}
	}
	for dst, src := range links {
		if _, err := link.Link(src, dst); err != nil {
			return err
		}
		// Record the link in state so pruning sees de-declared skills later.
		st.Set("claude", "skill."+filepath.Base(dst), dst+" -> "+src, secret.Hash(dst+" -> "+src))
	}
	return nil
}

func readStandardized(path string) ([]byte, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return jsonutil.Standardize(nil)
	}
	if err != nil {
		return nil, err
	}
	doc, err := jsonutil.Standardize(b)
	if err != nil {
		return nil, err
	}
	if err := jsonutil.ObjectRoot(doc); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return doc, nil
}
