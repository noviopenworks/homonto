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

func TestDoctorReportsSkillLinkState(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	os.WriteFile(filepath.Join(repo, "homonto.toml"), []byte("[skills]\nown=[\"graphify\"]\n"), 0o644)
	content := filepath.Join(repo, "content")
	os.MkdirAll(filepath.Join(content, "skills", "graphify"), 0o755)

	build := func() *Engine {
		e, err := Build(filepath.Join(repo, "homonto.toml"), home, content)
		if err != nil {
			t.Fatal(err)
		}
		return e
	}

	// content present but no symlink yet -> not linked
	lines := strings.Join(build().Doctor(), "\n")
	if !strings.Contains(lines, `skill "graphify" content present, not linked`) {
		t.Fatalf("doctor should report unlinked skill:\n%s", lines)
	}

	// correct symlink -> linked
	dst := filepath.Join(home, ".claude", "skills", "graphify")
	os.MkdirAll(filepath.Dir(dst), 0o755)
	if err := os.Symlink(filepath.Join(content, "skills", "graphify"), dst); err != nil {
		t.Fatal(err)
	}
	lines = strings.Join(build().Doctor(), "\n")
	if !strings.Contains(lines, `ok: skill "graphify" linked`) {
		t.Fatalf("doctor should report linked skill:\n%s", lines)
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

// buildStatusEngine wires an engine over the given repo/home with a stubbed
// resolver so secret-free config applies without touching `pass`.
func buildStatusEngine(t *testing.T, repo, home string) *Engine {
	t.Helper()
	e, err := Build(filepath.Join(repo, "homonto.toml"), home, filepath.Join(repo, "content"))
	if err != nil {
		t.Fatal(err)
	}
	e.Resolver = &secret.Resolver{Getenv: os.Getenv, Pass: func(string) (string, error) { return "", nil }}
	return e
}

func TestStatusDetectsDriftAfterOutOfBandChange(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	os.WriteFile(filepath.Join(repo, "homonto.toml"), []byte("[settings.claude]\nmodel=\"opus\"\n"), 0o644)

	e := buildStatusEngine(t, repo, home)
	sets, _ := e.Plan()
	if err := e.Apply(sets); err != nil {
		t.Fatal(err)
	}

	// no drift right after apply
	if d, pending, _ := buildStatusEngine(t, repo, home).Status(); len(d) != 0 || pending != 0 {
		t.Fatalf("unexpected status after clean apply: drift=%v pending=%d", d, pending)
	}

	// change the managed key ON DISK, out of band
	sj := filepath.Join(home, ".claude", "settings.json")
	os.WriteFile(sj, []byte(`{"model":"sonnet"}`), 0o644)

	drift, pending, err := buildStatusEngine(t, repo, home).Status()
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(drift, "\n")
	if len(drift) == 0 || !strings.Contains(joined, "model") || !strings.Contains(joined, "drifted") {
		t.Fatalf("expected drift on model, got %v", drift)
	}
	// The drifted key is reported as drift, not as pending config work.
	if pending != 0 {
		t.Fatalf("a disk-drifted key must not also count as pending, got pending=%d", pending)
	}
}

// TestStatusConfigEditIsPendingNotDrift is the load-bearing negative: a pure
// CONFIG edit (desired changes, disk unchanged) must NOT be reported as disk
// drift — it is a pending config change awaiting apply. The old Plan-based
// Drift mis-reported this as drift; this proves the fix.
func TestStatusConfigEditIsPendingNotDrift(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	os.WriteFile(filepath.Join(repo, "homonto.toml"), []byte("[settings.claude]\nmodel=\"opus\"\n"), 0o644)

	e := buildStatusEngine(t, repo, home)
	sets, _ := e.Plan()
	if err := e.Apply(sets); err != nil {
		t.Fatal(err)
	}

	// Edit ONLY the config (desired), leaving the on-disk value untouched.
	os.WriteFile(filepath.Join(repo, "homonto.toml"), []byte("[settings.claude]\nmodel=\"sonnet\"\n"), 0o644)

	drift, pending, err := buildStatusEngine(t, repo, home).Status()
	if err != nil {
		t.Fatal(err)
	}
	if len(drift) != 0 {
		t.Fatalf("a pure config edit must not be reported as disk drift, got %v", drift)
	}
	if pending != 1 {
		t.Fatalf("a pure config edit must count as one pending change, got pending=%d", pending)
	}
}

// TestStatusReportsMissingManagedKey: a state-recorded key removed from disk out
// of band is reported as missing (and does not count toward pending).
func TestStatusReportsMissingManagedKey(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	os.WriteFile(filepath.Join(repo, "homonto.toml"), []byte("[settings.claude]\nmodel=\"opus\"\n"), 0o644)

	e := buildStatusEngine(t, repo, home)
	sets, _ := e.Plan()
	if err := e.Apply(sets); err != nil {
		t.Fatal(err)
	}

	// remove the managed key from disk out of band
	sj := filepath.Join(home, ".claude", "settings.json")
	os.WriteFile(sj, []byte(`{}`), 0o644)

	drift, pending, err := buildStatusEngine(t, repo, home).Status()
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(drift, "\n")
	if !strings.Contains(joined, "model") || !strings.Contains(joined, "missing") {
		t.Fatalf("deleted managed key must report as missing, got %v", drift)
	}
	if pending != 0 {
		t.Fatalf("a missing (drifted) key must not count as pending, got pending=%d", pending)
	}
}

func TestStatusCleanAfterApply(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	os.WriteFile(filepath.Join(repo, "homonto.toml"), []byte("[settings.claude]\nmodel=\"opus\"\n"), 0o644)

	e := buildStatusEngine(t, repo, home)
	sets, _ := e.Plan()
	if err := e.Apply(sets); err != nil {
		t.Fatal(err)
	}

	drift, pending, err := buildStatusEngine(t, repo, home).Status()
	if err != nil {
		t.Fatal(err)
	}
	if len(drift) != 0 || pending != 0 {
		t.Fatalf("clean apply must yield no drift and no pending, got drift=%v pending=%d", drift, pending)
	}
}
