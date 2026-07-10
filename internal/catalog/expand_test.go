package catalog

import (
	"strings"
	"testing"
	"testing/fstest"
)

// graphFS builds a catalog whose skill content always exists, so Load passes and
// tests can focus on the dependency graph. deps maps framework -> dep names;
// skills maps framework -> its own skill names; commands maps framework -> its
// own command names.
func graphFS(deps map[string][]string, skills, commands map[string][]string) fstest.MapFS {
	m := fstest.MapFS{"version.txt": {Data: []byte("0.1.0")}}
	for fw, sk := range skills {
		var b strings.Builder
		b.WriteString("name = \"" + fw + "\"\nversion = \"0.1.0\"\n")
		if d := deps[fw]; len(d) > 0 {
			b.WriteString("[dependencies]\nframeworks = [")
			for i, dep := range d {
				if i > 0 {
					b.WriteString(", ")
				}
				b.WriteString("\"" + dep + "\"")
			}
			b.WriteString("]\n")
		}
		b.WriteString("[skills]\n")
		for _, s := range sk {
			b.WriteString(s + " = \"skills/" + s + "\"\n")
			m["skills/"+s+"/SKILL.md"] = &fstest.MapFile{Data: []byte("x")}
		}
		if cs := commands[fw]; len(cs) > 0 {
			b.WriteString("[commands]\n")
			for _, cmd := range cs {
				b.WriteString(cmd + " = \"commands/" + cmd + ".md\"\n")
				m["commands/"+cmd+".md"] = &fstest.MapFile{Data: []byte("x")}
			}
		}
		m["frameworks/"+fw+"/framework.toml"] = &fstest.MapFile{Data: []byte(b.String())}
	}
	return m
}

func TestExpandTransitiveAndDedup(t *testing.T) {
	// comet -> superpowers, openspec; superpowers and openspec share "shared".
	c, err := Load(graphFS(
		map[string][]string{"comet": {"superpowers", "openspec"}},
		map[string][]string{
			"comet":       {"comet-open"},
			"superpowers": {"brainstorming", "shared"},
			"openspec":    {"openspec-new", "shared"},
		},
		nil,
	))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	got, err := c.Expand([]string{"comet"})
	if err != nil {
		t.Fatalf("expand: %v", err)
	}
	var names []string
	for _, e := range got {
		names = append(names, e.Name)
	}
	want := []string{"brainstorming", "comet-open", "openspec-new", "shared"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Fatalf("expanded (sorted, deduped) = %v, want %v", names, want)
	}
}

func TestExpandDetectsCycle(t *testing.T) {
	c, err := Load(graphFS(
		map[string][]string{"a": {"b"}, "b": {"a"}},
		map[string][]string{"a": {"sa"}, "b": {"sb"}},
		nil,
	))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	_, err = c.Expand([]string{"a"})
	if err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("expected cycle error, got %v", err)
	}
	if !strings.Contains(err.Error(), "a") || !strings.Contains(err.Error(), "b") {
		t.Fatalf("cycle error should name the chain, got %v", err)
	}
}

func TestExpandCommandsTransitiveAndDedup(t *testing.T) {
	c, err := Load(graphFS(
		map[string][]string{"comet": {"superpowers", "openspec"}},
		map[string][]string{"comet": {"s"}, "superpowers": {"s"}, "openspec": {"s"}},
		map[string][]string{
			"comet":       {"comet-cmd"},
			"superpowers": {"brainstorm-cmd", "shared-cmd"},
			"openspec":    {"openspec-cmd", "shared-cmd"},
		},
	))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	got, err := c.ExpandCommands([]string{"comet"})
	if err != nil {
		t.Fatalf("expand commands: %v", err)
	}
	var names []string
	for _, e := range got {
		names = append(names, e.Name)
	}
	want := []string{"brainstorm-cmd", "comet-cmd", "openspec-cmd", "shared-cmd"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Fatalf("expanded commands = %v, want %v", names, want)
	}
}

func TestExpandCommandsDetectsCycle(t *testing.T) {
	c, err := Load(graphFS(
		map[string][]string{"a": {"b"}, "b": {"a"}},
		map[string][]string{"a": {"sa"}, "b": {"sb"}},
		map[string][]string{"a": {"ca"}, "b": {"cb"}},
	))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if _, err := c.ExpandCommands([]string{"a"}); err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("expected cycle error, got %v", err)
	}
}
