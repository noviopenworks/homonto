package cli

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/noviopenworks/homonto/internal/subagentpath"
)

// snapshotTree returns the sorted set of file paths (recursively) under root, or
// an empty slice when root does not exist. Used to assert doctor writes nothing.
func snapshotTree(t *testing.T, root string) []string {
	t.Helper()
	var files []string
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	sort.Strings(files)
	return files
}

// TestAgentsDoctorHealthy: a declared, installed, undrifted copy agent yields a
// clean bill of health.
func TestAgentsDoctorHealthy(t *testing.T) {
	home := t.TempDir()
	cfg, _ := addWorkspace(t, copyAgentTOML, map[string]string{"rev": "# rev\n"})

	if out, err := runCmd(t, home, "", "agents", "add", "rev", "--config", cfg); err != nil {
		t.Fatalf("agents add: %v\n%s", err, out)
	}

	out, err := runCmd(t, home, "", "agents", "doctor", "--config", cfg)
	if err != nil {
		t.Fatalf("healthy doctor must succeed, got err %v\n%s", err, out)
	}
	if out != "healthy\n" {
		t.Fatalf("healthy doctor must print exactly \"healthy\", got:\n%q", out)
	}
}

// TestAgentsDoctorDeclaredNotInstalled: a declared agent with no lockfile record
// is reported as not installed and exits non-zero.
func TestAgentsDoctorDeclaredNotInstalled(t *testing.T) {
	home := t.TempDir()
	cfg, _ := addWorkspace(t, copyAgentTOML, map[string]string{"rev": "# rev\n"})

	out, err := runCmd(t, home, "", "agents", "doctor", "--config", cfg)
	if err == nil {
		t.Fatalf("declared-but-not-installed must fail, got:\n%s", out)
	}
	if !strings.Contains(out, "rev") || !strings.Contains(out, "not installed") {
		t.Fatalf("must report rev not installed, got:\n%s", out)
	}
}

// TestAgentsDoctorOrphan: a lockfile agent no longer declared in the config is
// reported as an orphan and exits non-zero.
func TestAgentsDoctorOrphan(t *testing.T) {
	home := t.TempDir()
	cfg, _ := addWorkspace(t, copyAgentTOML, map[string]string{"rev": "# rev\n"})

	if out, err := runCmd(t, home, "", "agents", "add", "rev", "--config", cfg); err != nil {
		t.Fatalf("agents add: %v\n%s", err, out)
	}

	// Rewrite the same config (same dir => same .homonto lockfile) without rev.
	if err := os.WriteFile(cfg, []byte("\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCmd(t, home, "", "agents", "doctor", "--config", cfg)
	if err == nil {
		t.Fatalf("orphan install must fail, got:\n%s", out)
	}
	if !strings.Contains(out, "rev") || !strings.Contains(out, "orphan") {
		t.Fatalf("must report rev as orphan, got:\n%s", out)
	}
}

// TestAgentsDoctorSourceDrift: an installed local agent whose source file changed
// since install is reported as source drift and exits non-zero.
func TestAgentsDoctorSourceDrift(t *testing.T) {
	home := t.TempDir()
	cfg, cfgDir := addWorkspace(t, copyAgentTOML, map[string]string{"rev": "# rev\n"})

	if out, err := runCmd(t, home, "", "agents", "add", "rev", "--config", cfg); err != nil {
		t.Fatalf("agents add: %v\n%s", err, out)
	}

	// Change the provider source file after install.
	srcPath := filepath.Join(cfgDir, "homonto", "agents", "rev.md")
	if err := os.WriteFile(srcPath, []byte("# rev CHANGED\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCmd(t, home, "", "agents", "doctor", "--config", cfg)
	if err == nil {
		t.Fatalf("source drift must fail, got:\n%s", out)
	}
	if !strings.Contains(out, "source changed") {
		t.Fatalf("must report source changed, got:\n%s", out)
	}
}

// TestAgentsDoctorLocalEditIsHealthy: in the three-way-merge model a copy-mode
// install whose on-disk file was locally edited (but whose source is unchanged)
// is a normal, mergeable state — doctor does NOT flag it and exits 0.
func TestAgentsDoctorLocalEditIsHealthy(t *testing.T) {
	home := t.TempDir()
	cfg, _ := addWorkspace(t, copyAgentTOML, map[string]string{"rev": "# rev\n"})

	if out, err := runCmd(t, home, "", "agents", "add", "rev", "--config", cfg); err != nil {
		t.Fatalf("agents add: %v\n%s", err, out)
	}

	dst := filepath.Join(subagentpath.Dir("claude", "user", home, ""), "rev.md")
	if err := os.WriteFile(dst, []byte("# locally edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCmd(t, home, "", "agents", "doctor", "--config", cfg)
	if err != nil {
		t.Fatalf("a local edit must be healthy in the merge model, got err %v\n%s", err, out)
	}
	if strings.Contains(out, "modified") {
		t.Fatalf("doctor must NOT report a local edit as modified, got:\n%s", out)
	}
	if out != "healthy\n" {
		t.Fatalf("doctor must print exactly \"healthy\", got:\n%q", out)
	}
}

// TestAgentsDoctorReportsPendingConflict: a <dst>.merged sidecar left by a
// conflicted `agents update` is reported as a pending conflict and exits non-zero.
func TestAgentsDoctorReportsPendingConflict(t *testing.T) {
	home := t.TempDir()
	cfg, _ := addWorkspace(t, copyAgentTOML, map[string]string{"rev": "# rev\n"})

	if out, err := runCmd(t, home, "", "agents", "add", "rev", "--config", cfg); err != nil {
		t.Fatalf("agents add: %v\n%s", err, out)
	}

	dst := filepath.Join(subagentpath.Dir("claude", "user", home, ""), "rev.md")
	sidecar := "<<<<<<< local\n# mine\n=======\n# theirs\n>>>>>>> source\n"
	if err := os.WriteFile(dst+".merged", []byte(sidecar), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCmd(t, home, "", "agents", "doctor", "--config", cfg)
	if err == nil {
		t.Fatalf("a pending conflict must fail, got:\n%s", out)
	}
	if !strings.Contains(out, "conflicted") || !strings.Contains(out, dst+".merged") {
		t.Fatalf("must report the target as conflicted pointing at %s.merged, got:\n%s", dst, out)
	}
}

// TestAgentsDoctorMissingOnDisk: a recorded install whose file was deleted is
// reported as missing and exits non-zero.
func TestAgentsDoctorMissingOnDisk(t *testing.T) {
	home := t.TempDir()
	cfg, _ := addWorkspace(t, copyAgentTOML, map[string]string{"rev": "# rev\n"})

	if out, err := runCmd(t, home, "", "agents", "add", "rev", "--config", cfg); err != nil {
		t.Fatalf("agents add: %v\n%s", err, out)
	}

	dst := filepath.Join(subagentpath.Dir("claude", "user", home, ""), "rev.md")
	if err := os.Remove(dst); err != nil {
		t.Fatal(err)
	}

	out, err := runCmd(t, home, "", "agents", "doctor", "--config", cfg)
	if err == nil {
		t.Fatalf("missing-on-disk must fail, got:\n%s", out)
	}
	if !strings.Contains(out, "missing") {
		t.Fatalf("must report missing, got:\n%s", out)
	}
}

// TestAgentsDoctorIsReadOnly: doctor creates and mutates no files under the
// config dir or the home agent dirs.
func TestAgentsDoctorIsReadOnly(t *testing.T) {
	home := t.TempDir()
	cfg, cfgDir := addWorkspace(t, copyAgentTOML, map[string]string{"rev": "# rev\n"})

	if out, err := runCmd(t, home, "", "agents", "add", "rev", "--config", cfg); err != nil {
		t.Fatalf("agents add: %v\n%s", err, out)
	}

	beforeCfg := snapshotTree(t, cfgDir)
	beforeHome := snapshotTree(t, home)

	if out, err := runCmd(t, home, "", "agents", "doctor", "--config", cfg); err != nil {
		t.Fatalf("healthy doctor must succeed, got %v\n%s", err, out)
	}

	if got := snapshotTree(t, cfgDir); !equalStrings(got, beforeCfg) {
		t.Fatalf("doctor must not change config-dir files:\nbefore %v\nafter  %v", beforeCfg, got)
	}
	if got := snapshotTree(t, home); !equalStrings(got, beforeHome) {
		t.Fatalf("doctor must not change home files:\nbefore %v\nafter  %v", beforeHome, got)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
