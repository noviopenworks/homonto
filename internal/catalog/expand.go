package catalog

import (
	"fmt"
	"sort"
	"strings"
)

// ExpandedSkill is one skill reached by framework expansion, tagged with the
// framework it originated from (for later plan-origin notes).
type ExpandedSkill struct {
	Name      string
	Framework string
}

// Expand returns the transitive, deduplicated set of skills reachable from the
// given framework names, sorted by skill name, or an error naming a dependency
// cycle. A skill reachable via two frameworks collapses to one entry keyed by
// its first-seen origin.
func (c *Catalog) Expand(frameworkNames []string) ([]ExpandedSkill, error) {
	const (
		white = 0
		grey  = 1
		black = 2
	)
	color := map[string]int{}
	skills := map[string]ExpandedSkill{}
	var stack []string

	var visit func(name string) error
	visit = func(name string) error {
		f, ok := c.frameworks[name]
		if !ok {
			return fmt.Errorf("catalog: unknown framework %q", name)
		}
		switch color[name] {
		case grey:
			return fmt.Errorf("catalog: framework dependency cycle: %s", strings.Join(append(stack, name), " -> "))
		case black:
			return nil
		}
		color[name] = grey
		stack = append(stack, name)
		for _, dep := range f.Dependencies {
			if err := visit(dep); err != nil {
				return err
			}
		}
		for skill := range f.Skills {
			if _, seen := skills[skill]; !seen {
				skills[skill] = ExpandedSkill{Name: skill, Framework: name}
			}
		}
		stack = stack[:len(stack)-1]
		color[name] = black
		return nil
	}

	for _, n := range frameworkNames {
		if err := visit(n); err != nil {
			return nil, err
		}
	}

	out := make([]ExpandedSkill, 0, len(skills))
	for _, s := range skills {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}
