package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/noviopenworks/homonto/internal/plan"
	"github.com/noviopenworks/homonto/internal/secret"
)

// TestProjectScopeEndToEnd exercises the compiled apply path at project scope:
// links land under the project root, a second apply is idempotent, status is
// clean, and switching back to user scope relocates the links (no orphan).
func TestProjectScopeEndToEnd(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	content := filepath.Join(repo, "content")
	os.MkdirAll(filepath.Join(content, "skills", "graphify"), 0o755)

	writeTOML := func(scope string) {
		os.WriteFile(filepath.Join(repo, "homonto.toml"),
			[]byte("[skills]\nscope=\""+scope+"\"\nown=[\"graphify\"]\n"), 0o644)
	}
	build := func() *Engine {
		e, err := Build(filepath.Join(repo, "homonto.toml"), home, content)
		if err != nil {
			t.Fatal(err)
		}
		e.Resolver = &secret.Resolver{Getenv: os.Getenv, Pass: func(string) (string, error) { return "", nil }}
		return e
	}

	projClaude := filepath.Join(repo, ".claude", "skills", "graphify")
	projOpen := filepath.Join(repo, ".opencode", "skills", "graphify")
	homeClaude := filepath.Join(home, ".claude", "skills", "graphify")

	// 1. Apply at project scope.
	writeTOML("project")
	e := build()
	sets, _ := e.Plan()
	if err := e.Apply(sets); err != nil {
		t.Fatal(err)
	}
	for _, dst := range []string{projClaude, projOpen} {
		if _, err := os.Readlink(dst); err != nil {
			t.Fatalf("project link missing: %s (%v)", dst, err)
		}
	}
	if _, err := os.Lstat(homeClaude); err == nil {
		t.Fatal("project scope must not link under home")
	}

	// 2. Idempotent + status clean.
	e2 := build()
	sets2, _ := e2.Plan()
	if plan.HasChanges(sets2) {
		t.Fatalf("project-scope apply not idempotent: %s", plan.Render(sets2))
	}
	drift, pending, err := build().Status()
	if err != nil {
		t.Fatal(err)
	}
	if len(drift) != 0 || pending != 0 {
		t.Fatalf("status not clean at project scope: drift=%v pending=%d", drift, pending)
	}

	// 3. Switch back to user scope: links relocate, project links pruned.
	writeTOML("user")
	e3 := build()
	sets3, _ := e3.Plan()
	if err := e3.Apply(sets3); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Readlink(homeClaude); err != nil {
		t.Fatalf("user link not created after switch: %v", err)
	}
	if _, err := os.Lstat(projClaude); err == nil {
		t.Fatal("project claude link must be pruned after switch — orphan left behind")
	}
	if _, err := os.Lstat(projOpen); err == nil {
		t.Fatal("project opencode link must be pruned after switch — orphan left behind")
	}
}
