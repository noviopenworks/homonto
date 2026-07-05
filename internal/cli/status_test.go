package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// applied seeds home+config, applies once, and returns the config path so the
// caller can drive status against a recorded state.
func applied(t *testing.T, home, config string) string {
	t.Helper()
	repo := t.TempDir()
	cfg := filepath.Join(repo, "homonto.toml")
	if err := os.WriteFile(cfg, []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := runCmd(t, home, "", "apply", "--yes", "--config", cfg); err != nil {
		t.Fatalf("seed apply: %v\n%s", err, out)
	}
	return cfg
}

// A clean apply leaves status reporting no drift.
func TestStatusReportsNoDrift(t *testing.T) {
	home := t.TempDir()
	cfg := applied(t, home, "[settings.opencode]\nmodel=\"opus\"\n")

	out, err := runCmd(t, home, "", "status", "--config", cfg)
	if err != nil {
		t.Fatalf("status: %v\n%s", err, out)
	}
	if !strings.Contains(out, "No drift.") {
		t.Fatalf("clean status must report No drift., got:\n%s", out)
	}
}

// An out-of-band disk edit surfaces as a drift line.
func TestStatusReportsDriftLine(t *testing.T) {
	home := t.TempDir()
	cfg := applied(t, home, "[settings.opencode]\nmodel=\"opus\"\n")

	// Change the managed value on disk, out of band.
	oc := filepath.Join(home, ".config", "opencode", "opencode.jsonc")
	if err := os.WriteFile(oc, []byte(`{"model":"sonnet"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCmd(t, home, "", "status", "--config", cfg)
	if err != nil {
		t.Fatalf("status: %v\n%s", err, out)
	}
	if strings.Contains(out, "No drift.") {
		t.Fatalf("drifted disk must not report No drift., got:\n%s", out)
	}
	if !strings.Contains(out, "model") || !strings.Contains(out, "drifted") {
		t.Fatalf("status must list the drifted key, got:\n%s", out)
	}
}

// A pure config edit (desired changes, disk unchanged) is reported as pending,
// not drift.
func TestStatusReportsPendingConfigChange(t *testing.T) {
	home := t.TempDir()
	cfg := applied(t, home, "[settings.opencode]\nmodel=\"opus\"\n")

	// Edit only the config, leaving disk at the last-applied value.
	if err := os.WriteFile(cfg, []byte("[settings.opencode]\nmodel=\"sonnet\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCmd(t, home, "", "status", "--config", cfg)
	if err != nil {
		t.Fatalf("status: %v\n%s", err, out)
	}
	if strings.Contains(out, "drifted") || strings.Contains(out, "missing") {
		t.Fatalf("a pure config edit must not read as drift, got:\n%s", out)
	}
	if !strings.Contains(out, "config change(s) awaiting apply") {
		t.Fatalf("a pure config edit must report a pending change, got:\n%s", out)
	}
}
