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
	return &c, nil
}
