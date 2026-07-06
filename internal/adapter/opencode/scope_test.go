package opencode

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/noviopenworks/homonto/internal/adapter"
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

// TestScopeSwitchRelocatesLink: user -> project relocates the OpenCode link
// (home ~/.config/opencode/skills -> <proj>/.opencode/skills), pruning the old
// one; a foreign file at the inactive path is left untouched.
func TestScopeSwitchRelocatesLink(t *testing.T) {
	home := t.TempDir()
	os.MkdirAll(filepath.Join(home, ".config", "opencode"), 0o755)
	proj := t.TempDir()
	content := filepath.Join(proj, "content")
	os.MkdirAll(filepath.Join(content, "skills", "onto"), 0o755)
	st, _ := state.Load(t.TempDir())

	// 1. Apply at user scope.
	aUser := New(home, content).WithScope("user", proj)
	cUser := &config.Config{Skills: config.Skills{Scope: "user", Own: []string{"onto"}}}
	cs, err := aUser.Plan(cUser, st)
	if err != nil {
		t.Fatalf("user plan: %v", err)
	}
	if err := aUser.Apply(cs, noSecret(), st); err != nil {
		t.Fatalf("user apply: %v", err)
	}
	homeDst := filepath.Join(home, ".config", "opencode", "skills", "onto")
	if _, err := os.Readlink(homeDst); err != nil {
		t.Fatalf("user link missing: %v", err)
	}

	// 2. Switch to project scope; plan shows a relocate referencing home.
	aProj := New(home, content).WithScope("project", proj)
	cProj := &config.Config{Skills: config.Skills{Scope: "project", Own: []string{"onto"}}}
	cs2, err := aProj.Plan(cProj, st)
	if err != nil {
		t.Fatalf("project plan: %v", err)
	}
	var reloc *adapter.Change
	for i := range cs2.Changes {
		if cs2.Changes[i].Key == "skill.onto" {
			reloc = &cs2.Changes[i]
		}
	}
	if reloc == nil || reloc.Action != "update" || !strings.Contains(reloc.Old, homeDst) {
		t.Fatalf("expected relocate (update) referencing %q, got %+v", homeDst, reloc)
	}

	// 3. Apply: project link created, home link pruned.
	if err := aProj.Apply(cs2, noSecret(), st); err != nil {
		t.Fatalf("project apply: %v", err)
	}
	if _, err := os.Readlink(filepath.Join(proj, ".opencode", "skills", "onto")); err != nil {
		t.Fatalf("project link not created: %v", err)
	}
	if _, err := os.Lstat(homeDst); err == nil {
		t.Fatal("home link must be pruned after switch — orphan left behind")
	}
}

// TestRelocationPruneLeavesForeignFile: a real file at the inactive path is not
// removed and does not error the apply.
func TestRelocationPruneLeavesForeignFile(t *testing.T) {
	home := t.TempDir()
	os.MkdirAll(filepath.Join(home, ".config", "opencode"), 0o755)
	proj := t.TempDir()
	content := filepath.Join(proj, "content")
	os.MkdirAll(filepath.Join(content, "skills", "onto"), 0o755)
	// A user's own real file sits where a user-scope link would be — the prune
	// (this apply is project-scope) must leave it alone.
	homeSkills := filepath.Join(home, ".config", "opencode", "skills")
	os.MkdirAll(homeSkills, 0o755)
	foreign := filepath.Join(homeSkills, "onto")
	os.WriteFile(foreign, []byte("mine"), 0o644)

	st, _ := state.Load(t.TempDir())
	a := New(home, content).WithScope("project", proj)
	c := &config.Config{Skills: config.Skills{Scope: "project", Own: []string{"onto"}}}
	cs, err := a.Plan(c, st)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if err := a.Apply(cs, noSecret(), st); err != nil {
		t.Fatalf("apply must not error on a foreign file at the inactive path: %v", err)
	}
	if b, err := os.ReadFile(foreign); err != nil || string(b) != "mine" {
		t.Fatalf("foreign file was modified or removed: %v %q", err, string(b))
	}
}
