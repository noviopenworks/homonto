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

// ExpandedCommand is one command reached by framework expansion, tagged with
// the framework it originated from.
type ExpandedCommand struct {
	Name      string
	Framework string
}

// Expanded is one resource reached by framework expansion. It backs both
// Expand and ExpandCommands, which differ only in the resource map selected.
type Expanded struct {
	Name      string
	Framework string
}

// expandResources returns the transitive, deduplicated set of resources
// reachable from the given framework names — where sel picks a framework's
// resource map (Skills or Commands) — sorted by name, or an error naming a
// dependency cycle. A resource reachable via two frameworks collapses to one
// entry keyed by its first-seen origin. Cycle detection and dedup live here
// once, shared by Expand and ExpandCommands.
func (c *Catalog) expandResources(frameworkNames []string, sel func(Framework) map[string]string) ([]Expanded, error) {
	const (
		white = 0
		grey  = 1
		black = 2
	)
	color := map[string]int{}
	found := map[string]Expanded{}
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
		for res := range sel(f) {
			if _, seen := found[res]; !seen {
				found[res] = Expanded{Name: res, Framework: name}
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

	out := make([]Expanded, 0, len(found))
	for _, e := range found {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// Expand returns the transitive, deduplicated set of skills reachable from the
// given framework names, sorted by skill name, or an error naming a dependency
// cycle.
func (c *Catalog) Expand(frameworkNames []string) ([]ExpandedSkill, error) {
	res, err := c.expandResources(frameworkNames, func(f Framework) map[string]string { return f.Skills })
	if err != nil {
		return nil, err
	}
	out := make([]ExpandedSkill, len(res))
	for i, e := range res {
		out[i] = ExpandedSkill{Name: e.Name, Framework: e.Framework}
	}
	return out, nil
}

// ExpandCommands returns the transitive, deduplicated set of commands reachable
// from the given framework names, sorted by command name, or an error naming a
// dependency cycle.
func (c *Catalog) ExpandCommands(frameworkNames []string) ([]ExpandedCommand, error) {
	res, err := c.expandResources(frameworkNames, func(f Framework) map[string]string { return f.Commands })
	if err != nil {
		return nil, err
	}
	out := make([]ExpandedCommand, len(res))
	for i, e := range res {
		out[i] = ExpandedCommand{Name: e.Name, Framework: e.Framework}
	}
	return out, nil
}
