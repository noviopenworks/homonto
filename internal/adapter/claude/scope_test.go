package claude

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/noviopenworks/homonto/internal/adapter"
	"github.com/noviopenworks/homonto/internal/state"
)

// TestProjectScopeLinksUnderProjectRoot: with scope=project, skill links land
// under <projectRoot>/.claude/skills and nothing is created under home.
func TestProjectScopeLinksUnderProjectRoot(t *testing.T) {
	home := t.TempDir()
	os.WriteFile(filepath.Join(home, ".claude.json"), []byte(`{}`), 0o644)
	os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	os.WriteFile(filepath.Join(home, ".claude", "settings.json"), []byte(`{}`), 0o644)

	proj := t.TempDir()
	content := filepath.Join(proj, "content")
	os.MkdirAll(filepath.Join(content, "skills", "onto"), 0o755)

	a := New(home, content).WithProjectRoot(proj)
	st, _ := state.Load(t.TempDir())
	c := cfgWithSkills("project", "onto")

	cs, err := a.Plan(c, st)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if err := a.Apply(cs, resolver(), st); err != nil {
		t.Fatalf("apply: %v", err)
	}

	projDst := filepath.Join(proj, ".claude", "skills", "onto")
	if got, err := os.Readlink(projDst); err != nil || got != filepath.Join(content, "skills", "onto") {
		t.Fatalf("project link not created: %v %s", err, got)
	}
	if _, err := os.Lstat(filepath.Join(home, ".claude", "skills", "onto")); err == nil {
		t.Fatal("project scope must not create a link under home")
	}

	cs2, _ := a.Plan(c, st)
	for _, ch := range cs2.Changes {
		if ch.Action != "noop" {
			t.Fatalf("second plan must be all noop, got %s %s", ch.Action, ch.Key)
		}
	}
}

// TestRemoveAndSwitchLeavesNoOrphan (verify round 1, FINDING 1): removing a
// skill AND switching scope in one apply must prune the link at the location it
// actually occupies (the now-inactive scope), leaving no orphan.
func TestRemoveAndSwitchLeavesNoOrphan(t *testing.T) {
	home := t.TempDir()
	os.WriteFile(filepath.Join(home, ".claude.json"), []byte(`{}`), 0o644)
	os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	os.WriteFile(filepath.Join(home, ".claude", "settings.json"), []byte(`{}`), 0o644)
	proj := t.TempDir()
	content := filepath.Join(proj, "content")
	os.MkdirAll(filepath.Join(content, "skills", "foo"), 0o755)
	st, _ := state.Load(t.TempDir())

	aU := New(home, content).WithProjectRoot(proj)
	cU := cfgWithSkills("user", "foo")
	cs, _ := aU.Plan(cU, st)
	if err := aU.Apply(cs, resolver(), st); err != nil {
		t.Fatal(err)
	}
	homeDst := filepath.Join(home, ".claude", "skills", "foo")
	if _, err := os.Lstat(homeDst); err != nil {
		t.Fatalf("setup: user link missing: %v", err)
	}

	// Switch to project scope AND drop foo in the same apply.
	aP := New(home, content).WithProjectRoot(proj)
	cP := cfgWithSkills("project")
	cs2, _ := aP.Plan(cP, st)
	if err := aP.Apply(cs2, resolver(), st); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(homeDst); err == nil {
		t.Fatal("orphan: user-location link survived remove+switch")
	}
}

// TestSkillAdoptRebuildsState (verify round 1, FINDING 2): a correct-but-unrecorded
// skill link (e.g. after .homonto/state.json was lost) is adopted into state on
// apply without touching the link, so a skills-only config rebuilds state.
func TestSkillAdoptRebuildsState(t *testing.T) {
	home := t.TempDir()
	os.WriteFile(filepath.Join(home, ".claude.json"), []byte(`{}`), 0o644)
	os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	os.WriteFile(filepath.Join(home, ".claude", "settings.json"), []byte(`{}`), 0o644)
	content := t.TempDir()
	os.MkdirAll(filepath.Join(content, "skills", "foo"), 0o755)
	src := filepath.Join(content, "skills", "foo")
	dst := filepath.Join(home, ".claude", "skills", "foo")
	os.MkdirAll(filepath.Dir(dst), 0o755)
	if err := os.Symlink(src, dst); err != nil { // correct link already on disk
		t.Fatal(err)
	}

	st, _ := state.Load(t.TempDir()) // empty state — simulates lost state.json
	a := New(home, content)
	c := cfgWithSkills("user", "foo")

	cs, _ := a.Plan(c, st)
	var found *adapter.Change
	for i := range cs.Changes {
		if cs.Changes[i].Key == "skill.foo" {
			found = &cs.Changes[i]
		}
	}
	if found == nil || found.Action != "adopt" {
		t.Fatalf("expected adopt for skill.foo, got %+v", found)
	}
	if err := a.Apply(cs, resolver(), st); err != nil {
		t.Fatal(err)
	}
	if _, ok := st.Get("claude", "skill.foo"); !ok {
		t.Fatal("skill.foo not recorded in state after adopt")
	}
	if tgt, _ := os.Readlink(dst); tgt != src {
		t.Fatal("adopt must not change the on-disk link")
	}
	cs2, _ := a.Plan(c, st)
	for _, ch := range cs2.Changes {
		if ch.Action != "noop" {
			t.Fatalf("second plan must be noop after adopt, got %s %s", ch.Action, ch.Key)
		}
	}
}

// TestScopeSwitchRelocatesLink: switching user -> project relocates the link —
// plan shows a relocate (update) referencing the old location, apply creates the
// new link and prunes the old one, and the result is idempotent (no orphan).
func TestScopeSwitchRelocatesLink(t *testing.T) {
	home := t.TempDir()
	os.WriteFile(filepath.Join(home, ".claude.json"), []byte(`{}`), 0o644)
	os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	os.WriteFile(filepath.Join(home, ".claude", "settings.json"), []byte(`{}`), 0o644)

	proj := t.TempDir()
	content := filepath.Join(proj, "content")
	os.MkdirAll(filepath.Join(content, "skills", "onto"), 0o755)
	st, _ := state.Load(t.TempDir())

	// 1. Apply at user scope.
	aUser := New(home, content).WithProjectRoot(proj)
	cUser := cfgWithSkills("user", "onto")
	cs, err := aUser.Plan(cUser, st)
	if err != nil {
		t.Fatalf("user plan: %v", err)
	}
	if err := aUser.Apply(cs, resolver(), st); err != nil {
		t.Fatalf("user apply: %v", err)
	}
	homeDst := filepath.Join(home, ".claude", "skills", "onto")
	if _, err := os.Readlink(homeDst); err != nil {
		t.Fatalf("user link missing: %v", err)
	}

	// 2. Switch to project scope; plan must show a relocate referencing home.
	aProj := New(home, content).WithProjectRoot(proj)
	cProj := cfgWithSkills("project", "onto")
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
	if reloc == nil || reloc.Action != "update" {
		t.Fatalf("expected relocate (update) for skill.onto, got %+v", reloc)
	}
	if !strings.Contains(reloc.Old, homeDst) {
		t.Fatalf("relocate Old should reference the home location %q, got %q", homeDst, reloc.Old)
	}

	// 3. Apply the switch: project link created, home link pruned (no orphan).
	if err := aProj.Apply(cs2, resolver(), st); err != nil {
		t.Fatalf("project apply: %v", err)
	}
	projDst := filepath.Join(proj, ".claude", "skills", "onto")
	if _, err := os.Readlink(projDst); err != nil {
		t.Fatalf("project link not created: %v", err)
	}
	if _, err := os.Lstat(homeDst); err == nil {
		t.Fatal("home link must be pruned after switch — orphan left behind")
	}

	// 4. Idempotent.
	cs3, _ := aProj.Plan(cProj, st)
	for _, ch := range cs3.Changes {
		if ch.Action != "noop" {
			t.Fatalf("second plan after switch must be noop, got %s %s", ch.Action, ch.Key)
		}
	}
}
