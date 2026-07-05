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

// TestDoctorChecksOpenCodeSkillLink reproduces NEXT_AGENT gap #6: doctor
// verified only the Claude skill link, so a missing OpenCode link went
// unreported. Both tools' links must be checked, reported per tool.
func TestDoctorChecksOpenCodeSkillLink(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	os.WriteFile(filepath.Join(repo, "homonto.toml"), []byte("[skills]\nown=[\"graphify\"]\n"), 0o644)
	content := filepath.Join(repo, "content")
	os.MkdirAll(filepath.Join(content, "skills", "graphify"), 0o755)
	src := filepath.Join(content, "skills", "graphify")
	build := func() *Engine {
		e, err := Build(filepath.Join(repo, "homonto.toml"), home, content)
		if err != nil {
			t.Fatal(err)
		}
		return e
	}

	// Only the Claude link exists; the OpenCode link is missing.
	cl := filepath.Join(home, ".claude", "skills", "graphify")
	os.MkdirAll(filepath.Dir(cl), 0o755)
	if err := os.Symlink(src, cl); err != nil {
		t.Fatal(err)
	}
	lines := strings.Join(build().Doctor(), "\n")
	if !strings.Contains(lines, `ok: skill "graphify" linked (claude)`) {
		t.Fatalf("claude link should be reported ok per tool:\n%s", lines)
	}
	if !strings.Contains(lines, `skill "graphify" content present, not linked for opencode`) {
		t.Fatalf("doctor should warn about the missing opencode link:\n%s", lines)
	}

	// Add the OpenCode link too -> both report ok.
	ol := filepath.Join(home, ".config", "opencode", "skills", "graphify")
	os.MkdirAll(filepath.Dir(ol), 0o755)
	if err := os.Symlink(src, ol); err != nil {
		t.Fatal(err)
	}
	lines = strings.Join(build().Doctor(), "\n")
	if !strings.Contains(lines, `ok: skill "graphify" linked (opencode)`) {
		t.Fatalf("opencode link should be reported ok after linking:\n%s", lines)
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

// TestStatusDriftedKeyExcludedFromPendingWhileOthersCount proves the pending
// exclusion is specific to the drifted key: a key that is BOTH disk-drifted and
// config-edited is reported once as drift and NOT counted in pending, while a
// sibling key that is a pure config edit (disk == Applied) still counts. This
// locks in that pending tallies only OTHER config work, never the drifted key.
func TestStatusDriftedKeyExcludedFromPendingWhileOthersCount(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	os.WriteFile(filepath.Join(repo, "homonto.toml"),
		[]byte("[settings.claude]\nmodel=\"opus\"\ntheme=\"dark\"\n"), 0o644)

	e := buildStatusEngine(t, repo, home)
	sets, _ := e.Plan()
	if err := e.Apply(sets); err != nil {
		t.Fatal(err)
	}

	// model: change ON DISK to "sonnet" (disk drift) while editing desired to a
	// DIFFERENT "haiku" — so desired != disk != Applied, both disk-drifted AND a
	// config edit. theme: leave disk at "dark" (== Applied) but edit desired to
	// "light" — a pure config edit that must count as pending.
	sj := filepath.Join(home, ".claude", "settings.json")
	os.WriteFile(sj, []byte(`{"model":"sonnet","theme":"dark"}`), 0o644)
	os.WriteFile(filepath.Join(repo, "homonto.toml"),
		[]byte("[settings.claude]\nmodel=\"haiku\"\ntheme=\"light\"\n"), 0o644)

	drift, pending, err := buildStatusEngine(t, repo, home).Status()
	if err != nil {
		t.Fatal(err)
	}
	// model appears exactly once as drift; the pure config edit on theme does not.
	if len(drift) != 1 || !strings.Contains(drift[0], "model") || !strings.Contains(drift[0], "drifted") {
		t.Fatalf("expected exactly one drift entry for model, got %v", drift)
	}
	if strings.Contains(strings.Join(drift, "\n"), "theme") {
		t.Fatalf("a pure config edit on theme must not be reported as drift, got %v", drift)
	}
	// pending counts only the non-drifted config edit (theme); the drifted model
	// key is excluded even though it is also a config change.
	if pending != 1 {
		t.Fatalf("pending must count only the non-drifted config edit, got pending=%d", pending)
	}
}

// TestStatusSkipsErroredAdapterButReportsOther proves a per-adapter drift-scan
// failure is isolated: a malformed ~/.claude.json makes the Claude adapter's
// Plan and ObserveHashes both fail, so it is skipped with a warning, while a
// genuine OpenCode drift is still reported — no false "No drift".
func TestStatusSkipsErroredAdapterButReportsOther(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	os.WriteFile(filepath.Join(repo, "homonto.toml"),
		[]byte("[settings.opencode]\ntheme=\"dark\"\n"), 0o644)

	e := buildStatusEngine(t, repo, home)
	sets, _ := e.Plan()
	if err := e.Apply(sets); err != nil {
		t.Fatal(err)
	}

	// Drift the OpenCode setting on disk out of band.
	oc := filepath.Join(home, ".config", "opencode", "opencode.jsonc")
	os.WriteFile(oc, []byte(`{"theme":"light"}`), 0o644)
	// Corrupt ~/.claude.json so the Claude adapter cannot parse it: both its Plan
	// and ObserveHashes fail, exercising the skip-with-warning path.
	os.WriteFile(filepath.Join(home, ".claude.json"), []byte(`{ not json`), 0o644)

	e2 := buildStatusEngine(t, repo, home)
	drift, _, err := e2.Status()
	if err != nil {
		t.Fatal(err)
	}
	// The broken Claude adapter is reported as a warning, not a hard failure.
	if len(e2.Warnings) == 0 || !strings.Contains(strings.Join(e2.Warnings, "\n"), "claude") {
		t.Fatalf("expected a warning naming the skipped claude adapter, got %v", e2.Warnings)
	}
	// The healthy OpenCode adapter still reports its drift.
	joined := strings.Join(drift, "\n")
	if !strings.Contains(joined, "opencode") || !strings.Contains(joined, "theme") {
		t.Fatalf("opencode drift must still be reported despite the claude skip, got %v", drift)
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
