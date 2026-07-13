package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInit_AugmentsExistingGitignore(t *testing.T) {
	dir := t.TempDir()
	gi := filepath.Join(dir, ".gitignore")
	if err := os.WriteFile(gi, []byte("node_modules/\n"), 0o644); err != nil {
		t.Fatalf("seed gitignore: %v", err)
	}

	_, updated, err := Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}

	got, _ := os.ReadFile(gi)
	for _, want := range []string{"node_modules/", "/.homonto/", ".env"} {
		if !strings.Contains(string(got), want) {
			t.Errorf(".gitignore missing %q after Init; got %q", want, string(got))
		}
	}
	found := false
	for _, u := range updated {
		if strings.HasSuffix(u, ".gitignore") {
			found = true
		}
	}
	if !found {
		t.Errorf("updated %v should report the augmented .gitignore", updated)
	}
}
