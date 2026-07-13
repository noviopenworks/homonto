package ontocli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/noviopenworks/homonto/internal/ontostate"
)

func runOnto(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestSetIsolation_HappyPath_WritesField(t *testing.T) {
	root := prepWorkspace(t)
	seedChange(t, root, "c", "build")

	if _, err := runOnto(t, "set", "isolation", "c", "worktree", "--dir", root); err != nil {
		t.Fatalf("set isolation: %v", err)
	}
	st, err := ontostate.LoadChange(filepath.Join(root, "docs", "changes", "c"))
	if err != nil {
		t.Fatalf("LoadChange: %v", err)
	}
	if st.Isolation != "worktree" {
		t.Errorf("Isolation = %q, want worktree", st.Isolation)
	}
}

func TestSetIsolation_BadValue_RejectedNoWrite(t *testing.T) {
	root := prepWorkspace(t)
	seedChange(t, root, "c", "build")

	out, err := runOnto(t, "set", "isolation", "c", "vm", "--dir", root)
	if err == nil {
		t.Fatal("set isolation vm succeeded, want rejection")
	}
	if !strings.Contains(out+err.Error(), "isolation") {
		t.Errorf("error = %q, want it to name the field", err)
	}
	st, _ := ontostate.LoadChange(filepath.Join(root, "docs", "changes", "c"))
	if st.Isolation != "" {
		t.Errorf("Isolation = %q, want unchanged empty after rejected write", st.Isolation)
	}
}

func TestSetEnumSetters_HappyPaths(t *testing.T) {
	root := prepWorkspace(t)
	seedChange(t, root, "c", "build")

	cases := []struct {
		field, value string
		read         func(ontostate.State) string
	}{
		{"build-mode", "subagent", func(s ontostate.State) string { return s.BuildMode }},
		{"tdd-mode", "tdd", func(s ontostate.State) string { return s.TDDMode }},
		{"verify-scale", "full", func(s ontostate.State) string { return s.Verify.Scale }},
		{"verify-result", "pass", func(s ontostate.State) string { return s.Verify.Result }},
	}
	for _, tc := range cases {
		if _, err := runOnto(t, "set", tc.field, "c", tc.value, "--dir", root); err != nil {
			t.Fatalf("set %s: %v", tc.field, err)
		}
		st, _ := ontostate.LoadChange(filepath.Join(root, "docs", "changes", "c"))
		if got := tc.read(st); got != tc.value {
			t.Errorf("after set %s: got %q, want %q", tc.field, got, tc.value)
		}
	}
}

func TestSetCloseMerged_SetsTrueIdempotently(t *testing.T) {
	root := prepWorkspace(t)
	seedChange(t, root, "c", "close")

	for i := 0; i < 2; i++ { // idempotent: running twice is fine
		if _, err := runOnto(t, "set", "close-merged", "c", "--dir", root); err != nil {
			t.Fatalf("set close-merged (run %d): %v", i, err)
		}
	}
	st, _ := ontostate.LoadChange(filepath.Join(root, "docs", "changes", "c"))
	if !st.Close.Merged {
		t.Errorf("Close.Merged = false, want true")
	}
}

func TestSetDirective_StoresVerbatim(t *testing.T) {
	root := prepWorkspace(t)
	seedChange(t, root, "c", "build")

	const text = "ship without re-asking the isolation gate"
	if _, err := runOnto(t, "set", "directive", "c", text, "--dir", root); err != nil {
		t.Fatalf("set directive: %v", err)
	}
	st, _ := ontostate.LoadChange(filepath.Join(root, "docs", "changes", "c"))
	if st.Directive != text {
		t.Errorf("Directive = %q, want %q", st.Directive, text)
	}
}

func TestSetDirective_EmptyRejected(t *testing.T) {
	root := prepWorkspace(t)
	seedChange(t, root, "c", "build")

	if _, err := runOnto(t, "set", "directive", "c", "", "--dir", root); err == nil {
		t.Fatal("empty directive accepted, want rejection")
	}
}
