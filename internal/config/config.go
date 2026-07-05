package config

import (
	"fmt"
	"os"
	"path/filepath"
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

type Skills struct {
	Own []string `toml:"own"`
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
	MCPs     map[string]MCP `toml:"mcps"`
	Skills   Skills         `toml:"skills"`
	Plugins  Plugins        `toml:"plugins"`
	Settings Settings       `toml:"settings"`
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
	// Skill names become symlink path components under $HOME; anything but a
	// bare directory name (traversal, separators, "..") is rejected up front.
	for _, n := range c.Skills.Own {
		if n == "" || n == "." || n == ".." || strings.ContainsAny(n, `/\`) || n != filepath.Base(n) {
			return nil, fmt.Errorf("parse config: skills.own entry %q is not a plain directory name", n)
		}
	}
	// Every other name becomes a key written into a tool's JSON file. sjson
	// treats index-like segments ("0", "-1") as array positions, silently
	// turning the containing object into a JSON ARRAY; empty names address
	// nothing. Reject both up front with the offending entry named.
	for name := range c.MCPs {
		if err := validateKey("mcps", name); err != nil {
			return nil, err
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
	for k := range c.Settings.Claude {
		if err := validateKey("settings.claude", k); err != nil {
			return nil, err
		}
	}
	for k := range c.Settings.OpenCode {
		if err := validateKey("settings.opencode", k); err != nil {
			return nil, err
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
