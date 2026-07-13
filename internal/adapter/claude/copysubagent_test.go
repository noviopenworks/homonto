package claude

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/state"
)

// TestCopyModeSubagentProjection exercises the full copy-mode subagent lifecycle
// through Plan+Apply: create a real file (not a symlink), idempotent re-apply,
// source-change update, local-edit backup+overwrite, and de-declare prune.
func TestCopyModeSubagentProjection(t *testing.T) {
	home := t.TempDir()
	content := t.TempDir()
	saDir := filepath.Join(content, "subagents")
	if err := os.MkdirAll(saDir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(saDir, "rev.md")
	writeFile(t, src, "v1")

	cfg := &config.Config{Subagents: map[string]config.Subagent{
		"rev": {Source: "local:rev", Scope: "user", Mode: "copy", Targets: []string{"claude"}},
	}}
	a := New(home, content)
	st, _ := state.Load(t.TempDir())
	dst := filepath.Join(home, ".claude", "agents", "rev.md")

	apply := func() {
		t.Helper()
		cs, err := a.Plan(cfg, st)
		if err != nil {
			t.Fatalf("plan: %v", err)
		}
		if err := a.Apply(cfg, cs, resolver(), st); err != nil {
			t.Fatalf("apply: %v", err)
		}
	}

	// 1. create — a real file (not a symlink), recorded in state.
	apply()
	fi, err := os.Lstat(dst)
	if err != nil {
		t.Fatalf("copy file not created: %v", err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		t.Fatal("copy-mode must write a real file, not a symlink")
	}
	if b, _ := os.ReadFile(dst); string(b) != "v1" {
		t.Fatalf("content = %q, want v1", b)
	}
	if _, ok := st.Get("claude", "subagentcopy.rev"); !ok {
		t.Fatal("subagentcopy.rev not recorded in state")
	}

	// 2. idempotent — re-plan emits no create/update for the copy subagent.
	cs2, _ := a.Plan(cfg, st)
	for _, c := range cs2.Changes {
		if c.Key == "subagentcopy.rev" {
			t.Fatalf("re-plan not idempotent: unexpected %+v", c)
		}
	}

	// 3. source change → update.
	writeFile(t, src, "v2")
	apply()
	if b, _ := os.ReadFile(dst); string(b) != "v2" {
		t.Fatalf("update: content = %q, want v2", b)
	}

	// 4. local edit → back it up to .bak, then overwrite with the new source.
	writeFile(t, dst, "user edit")
	writeFile(t, src, "v3")
	apply()
	if b, _ := os.ReadFile(dst); string(b) != "v3" {
		t.Fatalf("post-edit content = %q, want v3", b)
	}
	if b, _ := os.ReadFile(dst + ".bak"); string(b) != "user edit" {
		t.Fatalf("%s.bak = %q, want the preserved user edit", dst, b)
	}

	// 5. de-declare → prune the managed file and drop its state.
	empty := &config.Config{}
	csEmpty, err := a.Plan(empty, st)
	if err != nil {
		t.Fatalf("plan(empty): %v", err)
	}
	if err := a.Apply(empty, csEmpty, resolver(), st); err != nil {
		t.Fatalf("apply(empty): %v", err)
	}
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Fatal("de-declared copy subagent was not pruned")
	}
	if _, ok := st.Get("claude", "subagentcopy.rev"); ok {
		t.Fatal("subagentcopy.rev state not cleared after prune")
	}
}

func writeFile(t *testing.T, p, s string) {
	t.Helper()
	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}
