package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/noviopenworks/homonto/internal/schema"
	toml "github.com/pelletier/go-toml/v2"
)

// decode parses the raw TOML and enforces the schema-version forward-safety
// guard. It is the first config-loading phase.
func decode(data []byte) (*Config, error) {
	var c Config
	if err := toml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	// Forward-safety: refuse a config from a newer schema before any adapter,
	// plan, or apply logic runs, so an older binary never silently mis-applies
	// fields it does not understand (TOML unmarshal drops unknown keys). Absent/0
	// is a legacy config, treated as the current version.
	if c.SchemaVersion > CurrentConfigSchemaVersion {
		return nil, fmt.Errorf("parse config: unknown config schema version %d (this binary supports up to %d) — upgrade homonto: %w", c.SchemaVersion, CurrentConfigSchemaVersion, schema.ErrTooNew)
	}
	return &c, nil
}

// migrate folds legacy declaration forms into their current equivalents.
func migrate(c *Config) {
	// Option C: the imperative [agents.<name>] model is superseded by the
	// declarative [subagents.<name>] one. Fold every declared agent into an
	// equivalent copy-mode subagent (a declared [agents.X] wins over an explicit
	// [subagents.X] of the same name) and drop the agents table, so [agents.X]
	// still parses but is now projected by `apply` like any other subagent.
	if len(c.Agents) > 0 {
		if c.Subagents == nil {
			c.Subagents = map[string]Subagent{}
		}
		for name, ag := range c.Agents {
			mode := ag.Mode
			if mode == "" && strings.HasPrefix(ag.Source, "builtin:") {
				mode = "copy" // builtin agents had no linkable path — copy-only
			}
			// [agents.X] wins the DECLARATION, but a same-named [subagents.X]'s
			// per-tool model blocks are tuning, which [agents.X] has no syntax
			// for — carry them over instead of silently deleting them.
			prev := c.Subagents[name]
			c.Subagents[name] = Subagent{
				Source:   ag.Source,
				Scope:    "user", // agents installed at user scope
				Mode:     mode,
				Version:  ag.Version,
				Targets:  ag.Targets,
				Claude:   prev.Claude,
				OpenCode: prev.OpenCode,
			}
		}
		c.Agents = nil
	}
}

// normalize applies defaulting so downstream projection sees concrete values.
func normalize(c *Config) {
	// Subagents default to project scope when omitted (skills and commands still
	// require an explicit scope). Normalize before validation so downstream
	// projection sees a concrete scope. Model-route values are whitespace-trimmed
	// here too: validation used to trim while the render did not, so
	// `model = "opus "` passed the alias check and then missed the alias map at
	// render, silently dropping its variant.
	trimRoute := func(r ModelRoute) ModelRoute {
		return ModelRoute{
			Model:   strings.TrimSpace(r.Model),
			Effort:  strings.TrimSpace(r.Effort),
			Variant: strings.TrimSpace(r.Variant),
		}
	}
	for name, r := range c.Subagents {
		if r.Scope == "" {
			r.Scope = "project"
		}
		r.Claude = trimRoute(r.Claude)
		r.OpenCode = trimRoute(r.OpenCode)
		c.Subagents[name] = r
	}
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	c, err := decode(data)
	if err != nil {
		return nil, err
	}
	migrate(c)
	normalize(c)
	if err := validate(c); err != nil {
		return nil, err
	}
	if abs, err := filepath.Abs(filepath.Dir(path)); err == nil {
		c.baseDir = abs
	} else {
		c.baseDir = filepath.Dir(path)
	}
	return c, nil
}
