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
}

// Catalog is the loaded, indexed catalog.
type Catalog struct {
	fsys       fs.FS
	frameworks map[string]Framework
	skills     map[string]string // skill name -> catalog-relative path (global index)
	version    string
}

type frameworkTOML struct {
	Name         string `toml:"name"`
	Version      string `toml:"version"`
	Description  string `toml:"description"`
	Dependencies struct {
		Frameworks []string `toml:"frameworks"`
	} `toml:"dependencies"`
	Skills map[string]string `toml:"skills"`
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
	}
	vb, err := fs.ReadFile(fsys, "version.txt")
	if err != nil {
		return nil, fmt.Errorf("catalog: read version.txt: %w", err)
	}
	c.version = strings.TrimSpace(string(vb))

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
		c.frameworks[dir] = Framework{
			Name:         ft.Name,
			Version:      ft.Version,
			Description:  ft.Description,
			Dependencies: ft.Dependencies.Frameworks,
			Skills:       ft.Skills,
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
