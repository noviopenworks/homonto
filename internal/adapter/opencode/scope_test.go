package opencode

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/state"
)

// TestProjectScopeLinksUnderProjectRoot: with scope=project, skill links land
// under <projectRoot>/.opencode/skills and nothing is created under home's
// ~/.config/opencode/skills.
func TestProjectScopeLinksUnderProjectRoot(t *testing.T) {
	home := t.TempDir()
	os.MkdirAll(filepath.Join(home, ".config", "opencode"), 0o755)

	proj := t.TempDir()
	content := filepath.Join(proj, "content")
	os.MkdirAll(filepath.Join(content, "skills", "onto"), 0o755)

	a := New(home, content).WithScope("project", proj)
	st, _ := state.Load(t.TempDir())
	c := &config.Config{Skills: config.Skills{Scope: "project", Own: []string{"onto"}}}

	cs, err := a.Plan(c, st)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if err := a.Apply(cs, noSecret(), st); err != nil {
		t.Fatalf("apply: %v", err)
	}

	projDst := filepath.Join(proj, ".opencode", "skills", "onto")
	if got, err := os.Readlink(projDst); err != nil || got != filepath.Join(content, "skills", "onto") {
		t.Fatalf("project link not created: %v %s", err, got)
	}
	if _, err := os.Lstat(filepath.Join(home, ".config", "opencode", "skills", "onto")); err == nil {
		t.Fatal("project scope must not create a link under home")
	}

	cs2, _ := a.Plan(c, st)
	for _, ch := range cs2.Changes {
		if ch.Action != "noop" {
			t.Fatalf("second plan must be all noop, got %s %s", ch.Action, ch.Key)
		}
	}
}
