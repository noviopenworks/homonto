package scaffold

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitCreatesFilesAndSkipsExisting(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "homonto.toml"), []byte("# mine\n"), 0o644)

	created, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range created {
		if filepath.Base(p) == "homonto.toml" {
			t.Fatal("must not recreate existing homonto.toml")
		}
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "homonto.toml")); string(b) != "# mine\n" {
		t.Fatal("existing config overwritten")
	}
	if _, err := os.Stat(filepath.Join(dir, ".gitignore")); err != nil {
		t.Fatal(".gitignore not created")
	}
	if _, err := os.Stat(filepath.Join(dir, "content", "skills")); err != nil {
		t.Fatal("content/skills not created")
	}
}
