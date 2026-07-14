package catalog

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	embedded "github.com/noviopenworks/homonto/catalog"
)

func matFS() fstest.MapFS {
	return fstest.MapFS{
		"version.txt": {Data: []byte("0.1.0")},
		"frameworks/sp/framework.toml": {Data: []byte(`name = "sp"
version = "0.1.0"
[skills]
brainstorming = "skills/brainstorming"
[commands]
demo-cmd = "commands/demo-cmd.md"
`)},
		"skills/brainstorming/SKILL.md":            {Data: []byte("top")},
		"skills/brainstorming/references/notes.md": {Data: []byte("nested")},
		"commands/demo-cmd.md":                     {Data: []byte("command body")},
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

func TestMaterializeCommandsWritesFile(t *testing.T) {
	c, err := Load(matFS())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	dst := t.TempDir()
	if err := c.MaterializeCommands(dst, []string{"demo-cmd"}); err != nil {
		t.Fatalf("materialize commands: %v", err)
	}
	if b, _ := os.ReadFile(filepath.Join(dst, "demo-cmd.md")); string(b) != "command body" {
		t.Fatalf("demo-cmd.md = %q", b)
	}
}

func TestMaterializeCommandsOverwrites(t *testing.T) {
	c, _ := Load(matFS())
	dst := t.TempDir()
	os.WriteFile(filepath.Join(dst, "demo-cmd.md"), []byte("STALE"), 0o644)
	if err := c.MaterializeCommands(dst, []string{"demo-cmd"}); err != nil {
		t.Fatalf("materialize commands: %v", err)
	}
	if b, _ := os.ReadFile(filepath.Join(dst, "demo-cmd.md")); string(b) != "command body" {
		t.Fatalf("stale command not overwritten: %q", b)
	}
}

func TestMaterializeCommandsUnknownErrors(t *testing.T) {
	c, _ := Load(matFS())
	if err := c.MaterializeCommands(t.TempDir(), []string{"nope"}); err == nil {
		t.Fatal("expected error for unknown command")
	}
}

func TestMaterializeSubagentsWritesFileVerbatim(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	dst := t.TempDir()
	if err := c.MaterializeSubagents(dst, []string{"code-reviewer"}); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dst, "code-reviewer.md"))
	if err != nil {
		t.Fatalf("read materialized: %v", err)
	}
	sp, _ := c.SubagentPath("code-reviewer")
	want, err := fs.ReadFile(embedded.FS, sp)
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("materialized subagent is not byte-for-byte identical to catalog source")
	}
}

// code-reviewer ships a neutral homonto: access block, so materialize must also
// emit per-tool frontmatter variants: Claude gets a tools: allowlist, OpenCode a
// permission: map. The two cannot share one file (OpenCode rejects a string
// tools:), so each adapter links its own variant.
func TestMaterializeSubagentsWritesPerToolVariants(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	dst := t.TempDir()
	if err := c.MaterializeSubagents(dst, []string{"code-reviewer"}); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	claude, err := os.ReadFile(filepath.Join(dst, "code-reviewer.claude.md"))
	if err != nil {
		t.Fatalf("claude variant not written: %v", err)
	}
	if !bytes.Contains(claude, []byte("tools: Read, Grep, Glob")) || bytes.Contains(claude, []byte("permission:")) {
		t.Errorf("claude variant should carry a tools allowlist and no permission block:\n%s", claude)
	}
	oc, err := os.ReadFile(filepath.Join(dst, "code-reviewer.opencode.md"))
	if err != nil {
		t.Fatalf("opencode variant not written: %v", err)
	}
	if !bytes.Contains(oc, []byte("permission:")) || !bytes.Contains(oc, []byte("edit: deny")) || bytes.Contains(oc, []byte("tools:")) {
		t.Errorf("opencode variant should carry an edit-deny permission block and no tools string:\n%s", oc)
	}
	// The neutral block must not leak into either rendered variant.
	if bytes.Contains(claude, []byte("homonto:")) || bytes.Contains(oc, []byte("homonto:")) {
		t.Error("homonto: block leaked into a rendered variant")
	}
}

func TestMaterializeSubagentsUnknownErrors(t *testing.T) {
	c, _ := New()
	if err := c.MaterializeSubagents(t.TempDir(), []string{"nope"}); err == nil {
		t.Fatal("expected error for unknown subagent")
	}
}
