// Package catalog loads and expands the embedded framework/skill catalog.
// It is config-agnostic: it MUST NOT import internal/config.
package catalog

import (
	"fmt"
	"io/fs"
	"path"
	"strings"

	embedded "github.com/noviopenworks/homonto/catalog"
	toml "github.com/pelletier/go-toml/v2"
)

// Framework is one catalog framework's parsed metadata.
type Framework struct {
	Name         string
	Version      string
	Description  string
	Dependencies []string          // framework names
	Skills       map[string]string // skill name -> catalog-relative path ("skills/<n>")
	Commands     map[string]string // command name -> catalog-relative path ("commands/<n>.md")
	Subagents    map[string]string // subagent name -> catalog-relative path ("subagents/<n>.md")
}

// Catalog is the loaded, indexed catalog.
type Catalog struct {
	fsys       fs.FS
	frameworks map[string]Framework
	skills     map[string]string // skill name -> catalog-relative path (global index)
	commands   map[string]string // command name -> catalog-relative path (global index)
	subagents  map[string]string // subagent name -> catalog-relative path (global index)
	version    string
}

// CurrentManifestSchemaVersion is the framework.toml manifest format version
// this binary supports. A manifest declaring a higher version is rejected
// fail-closed at load, so an older binary never silently half-reads a newer
// manifest (E1 phase-1 forward-safety, mirroring the config/state schema
// versions). Absent/0 is a legacy manifest, treated as the current version.
const CurrentManifestSchemaVersion = 1

type frameworkTOML struct {
	ManifestSchema int    `toml:"manifest_schema"`
	Name           string `toml:"name"`
	Version        string `toml:"version"`
	Description    string `toml:"description"`
	Dependencies   struct {
		Frameworks []string `toml:"frameworks"`
	} `toml:"dependencies"`
	Skills    map[string]string `toml:"skills"`
	Commands  map[string]string `toml:"commands"`
	Subagents map[string]string `toml:"subagents"`
}

// New loads the production catalog from the embedded filesystem.
func New() (*Catalog, error) { return Load(embedded.FS) }

// Load parses every frameworks/<name>/framework.toml in fsys, validates that
// each declared skill path exists and that a framework's name equals its
// directory, and reads version.txt (trimmed).
func Load(fsys fs.FS) (*Catalog, error) {
	c := &Catalog{
		fsys:       fsys,
		frameworks: map[string]Framework{},
		skills:     map[string]string{},
		commands:   map[string]string{},
		subagents:  map[string]string{},
	}
	vb, err := fs.ReadFile(fsys, "version.txt")
	if err != nil {
		return nil, fmt.Errorf("catalog: read version.txt: %w", err)
	}
	c.version = strings.TrimSpace(string(vb))

	// Loose (framework-agnostic) subagents: every "<n>.md" file directly under
	// subagents/ is indexed by base name, independent of any framework
	// declaring it. Unlike skills/commands, subagents are designed to include
	// standalone builtins (e.g. code-reviewer, codebase-explorer) referenced
	// directly by an explicit [subagents.X] config entry with no framework
	// home. The subagents/ directory is optional — fixtures/tests that don't
	// exercise subagents need not provide one.
	if entries, err := fs.ReadDir(fsys, "subagents"); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := strings.TrimSuffix(e.Name(), ".md")
			if name == e.Name() {
				continue // not a ".md" file
			}
			c.subagents[name] = path.Join("subagents", e.Name())
		}
	}

	dirs, err := fs.ReadDir(fsys, "frameworks")
	if err != nil {
		return nil, fmt.Errorf("catalog: read frameworks: %w", err)
	}
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		dir := d.Name()
		tp := path.Join("frameworks", dir, "framework.toml")
		b, err := fs.ReadFile(fsys, tp)
		if err != nil {
			return nil, fmt.Errorf("catalog: read %s: %w", tp, err)
		}
		var ft frameworkTOML
		if err := toml.Unmarshal(b, &ft); err != nil {
			return nil, fmt.Errorf("catalog: parse %s: %w", tp, err)
		}
		// Forward-safety: refuse a manifest from a newer schema before indexing
		// any of its resources, so an older binary never silently half-reads a
		// newer framework manifest. Absent/0 is a legacy manifest (current).
		if ft.ManifestSchema > CurrentManifestSchemaVersion {
			return nil, fmt.Errorf("catalog: framework %q manifest_schema %d is newer than this binary supports (up to %d) — upgrade homonto", dir, ft.ManifestSchema, CurrentManifestSchemaVersion)
		}
		if ft.Name != dir {
			return nil, fmt.Errorf("catalog: framework %q declares name %q; name must equal directory", dir, ft.Name)
		}
		for skill, sp := range ft.Skills {
			if _, err := fs.Stat(fsys, sp); err != nil {
				return nil, fmt.Errorf("catalog: framework %q skill %q path %q missing from catalog", dir, skill, sp)
			}
			if prev, ok := c.skills[skill]; ok && prev != sp {
				return nil, fmt.Errorf("catalog: skill %q mapped to both %q and %q", skill, prev, sp)
			}
			c.skills[skill] = sp
		}
		for command, cp := range ft.Commands {
			if _, err := fs.Stat(fsys, cp); err != nil {
				return nil, fmt.Errorf("catalog: framework %q command %q path %q missing from catalog", dir, command, cp)
			}
			if prev, ok := c.commands[command]; ok && prev != cp {
				return nil, fmt.Errorf("catalog: command %q mapped to both %q and %q", command, prev, cp)
			}
			c.commands[command] = cp
		}
		for subagent, sap := range ft.Subagents {
			if _, err := fs.Stat(fsys, sap); err != nil {
				return nil, fmt.Errorf("catalog: framework %q subagent %q path %q missing from catalog", dir, subagent, sap)
			}
			if prev, ok := c.subagents[subagent]; ok && prev != sap {
				return nil, fmt.Errorf("catalog: subagent %q mapped to both %q and %q", subagent, prev, sap)
			}
			c.subagents[subagent] = sap
		}
		c.frameworks[dir] = Framework{
			Name:         ft.Name,
			Version:      ft.Version,
			Description:  ft.Description,
			Dependencies: ft.Dependencies.Frameworks,
			Skills:       ft.Skills,
			Commands:     ft.Commands,
			Subagents:    ft.Subagents,
		}
	}

	// Loose (framework-agnostic) skills and commands: a skills/<dir> holding a
	// SKILL.md, or a commands/<n>.md file, not already claimed by a framework is
	// indexed by name so it installs as builtin:<name> with no framework home
	// (mirrors loose subagents). Framework declarations take precedence.
	if entries, err := fs.ReadDir(fsys, "skills"); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			name := e.Name()
			if _, ok := c.skills[name]; ok {
				continue // a framework already declares this skill
			}
			sp := path.Join("skills", name)
			if _, err := fs.Stat(fsys, path.Join(sp, "SKILL.md")); err != nil {
				continue // a skill directory must hold a SKILL.md
			}
			c.skills[name] = sp
		}
	}
	if entries, err := fs.ReadDir(fsys, "commands"); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := strings.TrimSuffix(e.Name(), ".md")
			if name == e.Name() {
				continue // not a ".md" file
			}
			if _, ok := c.commands[name]; ok {
				continue // a framework already declares this command
			}
			c.commands[name] = path.Join("commands", e.Name())
		}
	}
	return c, nil
}

// Version returns the catalog version string from version.txt.
func (c *Catalog) Version() string { return c.version }

// Framework returns the indexed framework and whether it exists.
func (c *Catalog) Framework(name string) (Framework, bool) {
	f, ok := c.frameworks[name]
	return f, ok
}

// SkillPath returns a skill's catalog-relative path ("skills/<n>") and whether
// it is known.
func (c *Catalog) SkillPath(name string) (string, bool) {
	p, ok := c.skills[name]
	return p, ok
}

// CommandPath returns a command's catalog-relative path ("commands/<n>.md") and
// whether it is known.
func (c *Catalog) CommandPath(name string) (string, bool) {
	p, ok := c.commands[name]
	return p, ok
}

// SubagentPath returns a subagent's catalog-relative path ("subagents/<n>.md")
// and whether it is known.
func (c *Catalog) SubagentPath(name string) (string, bool) {
	p, ok := c.subagents[name]
	return p, ok
}

// SubagentContent returns a builtin subagent/agent's content by name from the
// catalog filesystem, and whether the name is known.
func (c *Catalog) SubagentContent(name string) ([]byte, bool, error) {
	p, ok := c.subagents[name]
	if !ok {
		return nil, false, nil
	}
	b, err := fs.ReadFile(c.fsys, p)
	return b, true, err
}
