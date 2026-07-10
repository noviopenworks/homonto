package catalog

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func matFS() fstest.MapFS {
	return fstest.MapFS{
		"version.txt": {Data: []byte("0.1.0")},
		"frameworks/sp/framework.toml": {Data: []byte(`name = "sp"
version = "0.1.0"
[skills]
brainstorming = "skills/brainstorming"
`)},
		"skills/brainstorming/SKILL.md":            {Data: []byte("top")},
		"skills/brainstorming/references/notes.md": {Data: []byte("nested")},
	}
}

func TestMaterializeWritesNestedContent(t *testing.T) {
	c, err := Load(matFS())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	dst := t.TempDir()
	if err := c.Materialize(dst, []string{"brainstorming"}); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	if b, _ := os.ReadFile(filepath.Join(dst, "brainstorming", "SKILL.md")); string(b) != "top" {
		t.Fatalf("SKILL.md = %q", b)
	}
	if b, _ := os.ReadFile(filepath.Join(dst, "brainstorming", "references", "notes.md")); string(b) != "nested" {
		t.Fatalf("nested references/notes.md = %q", b)
	}
}

func TestMaterializeRemovesStaleOnUpgrade(t *testing.T) {
	c, err := Load(matFS())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	dst := t.TempDir()
	// Pre-seed a stale file that the new content does not include.
	os.MkdirAll(filepath.Join(dst, "brainstorming"), 0o755)
	os.WriteFile(filepath.Join(dst, "brainstorming", "STALE.md"), []byte("old"), 0o644)

	if err := c.Materialize(dst, []string{"brainstorming"}); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, "brainstorming", "STALE.md")); !os.IsNotExist(err) {
		t.Fatal("stale file survived materialization")
	}
}

func TestMaterializeUnknownSkillErrors(t *testing.T) {
	c, _ := Load(matFS())
	if err := c.Materialize(t.TempDir(), []string{"nope"}); err == nil {
		t.Fatal("expected error for unknown skill")
	}
}
