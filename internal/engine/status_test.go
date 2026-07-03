package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/noviopenworks/homonto/internal/secret"
)

func TestDoctorFlagsMissingSkillContent(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	os.WriteFile(filepath.Join(repo, "homonto.toml"), []byte("[skills]\nown=[\"ghost\"]\n"), 0o644)

	e, err := Build(filepath.Join(repo, "homonto.toml"), home, filepath.Join(repo, "content"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Join(e.Doctor(), "\n")
	if !strings.Contains(lines, "ghost") {
		t.Fatalf("doctor should flag missing skill 'ghost':\n%s", lines)
	}
}

func TestDoctorChecksToolConfigLocations(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	os.WriteFile(filepath.Join(repo, "homonto.toml"), []byte(""), 0o644)
	// Claude dir present, OpenCode dir absent
	os.MkdirAll(filepath.Join(home, ".claude"), 0o755)

	e, err := Build(filepath.Join(repo, "homonto.toml"), home, filepath.Join(repo, "content"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Join(e.Doctor(), "\n")
	if !strings.Contains(lines, ".claude") {
		t.Fatalf("doctor should mention the claude config location:\n%s", lines)
	}
	if !strings.Contains(lines, "opencode") {
		t.Fatalf("doctor should mention the opencode config location:\n%s", lines)
	}
}

func TestDriftDetectedAfterOutOfBandChange(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	os.WriteFile(filepath.Join(repo, "homonto.toml"), []byte("[settings.claude]\nmodel=\"opus\"\n"), 0o644)

	build := func() *Engine {
		e, err := Build(filepath.Join(repo, "homonto.toml"), home, filepath.Join(repo, "content"))
		if err != nil {
			t.Fatal(err)
		}
		e.Resolver = &secret.Resolver{Getenv: os.Getenv, Pass: func(string) (string, error) { return "", nil }}
		return e
	}

	e := build()
	sets, _ := e.Plan()
	if err := e.Apply(sets); err != nil {
		t.Fatal(err)
	}

	// no drift right after apply
	e2 := build()
	if d, _ := e2.Drift(); len(d) != 0 {
		t.Fatalf("unexpected drift after clean apply: %v", d)
	}

	// change the managed key out of band
	sj := filepath.Join(home, ".claude", "settings.json")
	os.WriteFile(sj, []byte(`{"model":"sonnet"}`), 0o644)

	e3 := build()
	d, _ := e3.Drift()
	if len(d) == 0 || !strings.Contains(strings.Join(d, "\n"), "model") {
		t.Fatalf("expected drift on model, got %v", d)
	}
}
