package catalog

import (
	"strings"
	"testing"
	"testing/fstest"
)

func fixtureFS() fstest.MapFS {
	return fstest.MapFS{
		"version.txt": {Data: []byte("0.1.0\n")},
		"frameworks/superpowers/framework.toml": {Data: []byte(`name = "superpowers"
version = "0.1.0"
description = "sp"
[skills]
brainstorming = "skills/brainstorming"
[commands]
demo-cmd = "commands/demo-cmd.md"
`)},
		"frameworks/comet/framework.toml": {Data: []byte(`name = "comet"
version = "0.1.0"
description = "cm"
[dependencies]
frameworks = ["superpowers"]
[skills]
comet = "skills/comet"
`)},
		"skills/brainstorming/SKILL.md": {Data: []byte("b")},
		"skills/comet/SKILL.md":         {Data: []byte("c")},
		"commands/demo-cmd.md":          {Data: []byte("d")},
	}
}

func TestLoadIndexesFrameworksAndVersion(t *testing.T) {
	c, err := Load(fixtureFS())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if c.Version() != "0.1.0" {
		t.Fatalf("version = %q", c.Version())
	}
	cm, ok := c.Framework("comet")
	if !ok {
		t.Fatal("comet not indexed")
	}
	if len(cm.Dependencies) != 1 || cm.Dependencies[0] != "superpowers" {
		t.Fatalf("comet deps = %v", cm.Dependencies)
	}
	if p, ok := c.SkillPath("brainstorming"); !ok || p != "skills/brainstorming" {
		t.Fatalf("brainstorming path = %q ok=%v", p, ok)
	}
}

func TestLoadRejectsMissingSkillPath(t *testing.T) {
	m := fixtureFS()
	delete(m, "skills/comet/SKILL.md") // now skills/comet has no entries -> path absent
	_, err := Load(m)
	if err == nil || !strings.Contains(err.Error(), "skills/comet") {
		t.Fatalf("expected missing-skill-path error, got %v", err)
	}
}

func TestLoadIndexesFrameworkCommands(t *testing.T) {
	c, err := Load(fixtureFS())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	sp, ok := c.Framework("superpowers")
	if !ok {
		t.Fatal("superpowers not indexed")
	}
	if sp.Commands["demo-cmd"] != "commands/demo-cmd.md" {
		t.Fatalf("superpowers commands = %v", sp.Commands)
	}
	if p, ok := c.CommandPath("demo-cmd"); !ok || p != "commands/demo-cmd.md" {
		t.Fatalf("demo-cmd path = %q ok=%v", p, ok)
	}
}

func TestLoadRejectsMissingCommandPath(t *testing.T) {
	m := fixtureFS()
	delete(m, "commands/demo-cmd.md")
	_, err := Load(m)
	if err == nil || !strings.Contains(err.Error(), "commands/demo-cmd.md") {
		t.Fatalf("expected missing-command-path error, got %v", err)
	}
}

func TestLoadRejectsNameDirMismatch(t *testing.T) {
	m := fixtureFS()
	m["frameworks/comet/framework.toml"] = &fstest.MapFile{Data: []byte(`name = "wrong"
version = "0.1.0"
[skills]
comet = "skills/comet"
`)}
	_, err := Load(m)
	if err == nil || !strings.Contains(err.Error(), "name") {
		t.Fatalf("expected name/dir mismatch error, got %v", err)
	}
}
