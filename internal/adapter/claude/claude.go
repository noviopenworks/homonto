package claude

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/noviopenworks/homonto/internal/adapter"
	"github.com/noviopenworks/homonto/internal/config"
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
// skills/commands/rules/agents.
func New(home, content string) *Adapter { return &Adapter{home: home, content: content} }

func (a *Adapter) Name() string { return "claude" }

func (a *Adapter) claudeJSON() string   { return filepath.Join(a.home, ".claude.json") }
func (a *Adapter) settingsJSON() string { return filepath.Join(a.home, ".claude", "settings.json") }

// desired returns managed key -> unresolved JSON-encoded desired value.
func (a *Adapter) desired(c *config.Config) map[string]string {
	out := map[string]string{}
	for name, m := range c.MCPs {
		if !contains(m.TargetsOrAll(), "claude") {
			continue
		}
		obj := map[string]any{"command": m.Command}
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
	for key, want := range a.desired(c) {
		disk, hasDisk := cur[key]
		e, inState := st.Get("claude", key)
		switch {
		case !hasDisk:
			cs.Changes = append(cs.Changes, adapter.Change{Action: "create", Key: key, New: want})
		case !secret.ContainsRef(want):
			if jsonutil.Canonical(disk) == jsonutil.Canonical(want) {
				cs.Changes = append(cs.Changes, adapter.Change{Action: "noop", Key: key})
			} else {
				cs.Changes = append(cs.Changes, adapter.Change{Action: "update", Key: key, Old: disk, New: want})
			}
		default: // secret-bearing key: never expose the on-disk resolved value
			if inState && e.Desired == want && e.Applied == secret.Hash(jsonutil.Canonical(disk)) {
				cs.Changes = append(cs.Changes, adapter.Change{Action: "noop", Key: key})
			} else {
				cs.Changes = append(cs.Changes, adapter.Change{Action: "update", Key: key, Old: adapter.SecretRedaction, New: want})
			}
		}
	}
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
	for _, c := range cs.Changes {
		if c.Action == "noop" {
			continue
		}
		resolved, err := res.Resolve(c.New)
		if err != nil {
			return err
		}
		var val any
		if err := json.Unmarshal([]byte(resolved), &val); err != nil {
			return err
		}
		switch {
		case hasPrefix(c.Key, "mcp."):
			mj, err = jsonutil.SetJSON(mj, "mcpServers."+trim(c.Key, "mcp."), val)
		case hasPrefix(c.Key, "setting."):
			sj, err = jsonutil.SetJSON(sj, trim(c.Key, "setting."), val)
		case hasPrefix(c.Key, "plugin."):
			sj, err = jsonutil.SetJSON(sj, "enabledPlugins."+trim(c.Key, "plugin."), val)
		}
		if err != nil {
			return err
		}
		// Store the unresolved form + a non-secret hash of the resolved value.
		st.Set("claude", c.Key, c.New, secret.Hash(jsonutil.Canonical(resolved)))
	}
	if err := writeAtomic(a.claudeJSON(), mj); err != nil {
		return err
	}
	if err := writeAtomic(a.settingsJSON(), sj); err != nil {
		return err
	}
	for _, name := range a.skills {
		src := filepath.Join(a.content, "skills", name)
		dst := filepath.Join(a.home, ".claude", "skills", name)
		if _, err := link.Link(src, dst); err != nil {
			return err
		}
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
	return jsonutil.Standardize(b)
}
