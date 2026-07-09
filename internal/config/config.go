package config

import (
	"fmt"
	"os"
	"sort"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
)

// MCP is a declared MCP server. Env values may hold unresolved ${...} tokens.
type MCP struct {
	Command []string          `toml:"command"`
	Env     map[string]string `toml:"env"`
	Targets []string          `toml:"targets"`
}

// TargetsOrAll returns the explicit targets, or all tools when none are set.
func (m MCP) TargetsOrAll() []string {
	if len(m.Targets) == 0 {
		return []string{"claude", "opencode"}
	}
	return m.Targets
}

type Resource struct {
	Source  string   `toml:"source"`
	Scope   string   `toml:"scope"`
	Targets []string `toml:"targets"`
}

func (r Resource) TargetsOrAll() []string {
	if len(r.Targets) == 0 {
		return []string{"claude", "opencode"}
	}
	return r.Targets
}

type NamedResource struct {
	Name     string
	Resource Resource
}

type ModelRoute struct {
	Model   string `toml:"model"`
	Effort  string `toml:"effort"`
	Variant string `toml:"variant"`
}

type ModelConfig struct {
	Claude   map[string]ModelRoute `toml:"claude"`
	OpenCode map[string]ModelRoute `toml:"opencode"`
}

type Plugins struct {
	Claude   []string `toml:"claude"`
	OpenCode []string `toml:"opencode"`
}

type Settings struct {
	Claude   map[string]any `toml:"claude"`
	OpenCode map[string]any `toml:"opencode"`
}

// Config is the tool-agnostic desired state parsed from homonto.toml.
type Config struct {
	MCPs       map[string]MCP      `toml:"mcps"`
	Frameworks map[string]Resource `toml:"frameworks"`
	Skills     map[string]Resource `toml:"skills"`
	Commands   map[string]Resource `toml:"commands"`
	Subagents  map[string]Resource `toml:"subagents"`
	Models     ModelConfig         `toml:"models"`
	Plugins    Plugins             `toml:"plugins"`
	Settings   Settings            `toml:"settings"`
}

func (c *Config) SkillEntriesForTool(tool string) []NamedResource {
	return entriesForTool(c.Skills, tool)
}

func (c *Config) EnabledModelTools() []string {
	seen := map[string]bool{}
	for _, resources := range []map[string]Resource{c.Frameworks, c.Commands, c.Subagents} {
		for _, r := range resources {
			for _, target := range r.TargetsOrAll() {
				seen[target] = true
			}
		}
	}
	out := make([]string, 0, len(seen))
	for tool := range seen {
		out = append(out, tool)
	}
	sort.Strings(out)
	return out
}

func entriesForTool(resources map[string]Resource, tool string) []NamedResource {
	var out []NamedResource
	for name, r := range resources {
		if containsString(r.TargetsOrAll(), tool) {
			out = append(out, NamedResource{Name: name, Resource: r})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func containsString(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}

// Load reads and parses a homonto.toml file into a Config.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var c Config
	if err := toml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	// Every other name becomes a key written into a tool's JSON file. sjson
	// treats index-like segments ("0", "-1") as array positions, silently
	// turning the containing object into a JSON ARRAY; empty names address
	// nothing. Reject both up front with the offending entry named.
	for name, m := range c.MCPs {
		if err := validateKey("mcps", name); err != nil {
			return nil, err
		}
		// An MCP with no command cannot project — both adapters would skip it,
		// so a declared server would silently do nothing. Fail fast instead.
		if len(m.Command) == 0 {
			return nil, fmt.Errorf("parse config: mcps entry %q has no command; an MCP server needs a command to run", name)
		}
		// A target that names no known tool matches no adapter, so the MCP is
		// projected nowhere — a silent typo. Only claude and opencode exist.
		for _, target := range m.Targets {
			if target != "claude" && target != "opencode" {
				return nil, fmt.Errorf("parse config: mcps entry %q targets unknown tool %q; valid targets are \"claude\" and \"opencode\"", name, target)
			}
		}
	}
	for _, p := range c.Plugins.Claude {
		if err := validateKey("plugins.claude", p); err != nil {
			return nil, err
		}
	}
	for _, p := range c.Plugins.OpenCode {
		if err := validateKey("plugins.opencode", p); err != nil {
			return nil, err
		}
	}
	// Settings keys that homonto itself manages in the same tool file would
	// collide with its own writes: claude projects plugins as `enabledPlugins`
	// into settings.json; opencode projects MCPs and plugins as the `mcp` and
	// `plugin` structures in opencode.jsonc. Reject those reserved names.
	//
	// `mcpServers` is reserved too: claude's current() deliberately skips that
	// settings.json key when reading managed values back (MCP servers are owned
	// via [mcps], projected into .claude.json). A settings.claude.mcpServers
	// value would be written on apply but never read back, so every plan would
	// re-propose it — a non-idempotent loop. Reject it up front instead.
	for k := range c.Settings.Claude {
		if err := validateKey("settings.claude", k); err != nil {
			return nil, err
		}
		if k == "enabledPlugins" {
			return nil, fmt.Errorf("parse config: settings.claude key %q is reserved (homonto manages plugins there); rename it", k)
		}
		if k == "mcpServers" {
			return nil, fmt.Errorf("parse config: settings.claude key %q is reserved (homonto manages MCP servers via [mcps]); declare the server under [mcps] instead", k)
		}
	}
	for k := range c.Settings.OpenCode {
		if err := validateKey("settings.opencode", k); err != nil {
			return nil, err
		}
		if k == "mcp" || k == "plugin" {
			return nil, fmt.Errorf("parse config: settings.opencode key %q is reserved (homonto manages %s there); rename it", k, k)
		}
	}
	return &c, nil
}

// validateKey rejects names unusable as literal JSON object keys: empty, or
// index-like (all digits, or "-" followed by digits — sjson array semantics).
func validateKey(kind, name string) error {
	if name == "" {
		return fmt.Errorf("parse config: %s entry %q is empty", kind, name)
	}
	if indexLike(name) {
		return fmt.Errorf("parse config: %s entry %q would be treated as a JSON array index and corrupt the target file; rename it", kind, name)
	}
	return nil
}

// indexLike reports whether sjson would treat name as an array index:
// all-digit ("0", "42") or "-" followed by digits ("-1", the append form).
func indexLike(name string) bool {
	t := strings.TrimPrefix(name, "-")
	if t == "" {
		return false // "-" alone is a plain key
	}
	for i := 0; i < len(t); i++ {
		if t[i] < '0' || t[i] > '9' {
			return false
		}
	}
	return true
}
