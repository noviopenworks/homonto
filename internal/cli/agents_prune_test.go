package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/noviopenworks/homonto/internal/agentlock"
	"github.com/noviopenworks/homonto/internal/subagentpath"
)

// claudeDst / opencodeDst return the user-dir install path for an agent+tool.
func claudeDst(home, name string) string {
	return filepath.Join(subagentpath.Dir("claude", "user", home, ""), name+".md")
}
func opencodeDst(home, name string) string {
	return filepath.Join(subagentpath.Dir("opencode", "user", home, ""), name+".md")
}

// TestAgentsPruneOrphanAgent: an agent recorded in the lockfile but no longer
// declared has its install files removed and its lockfile entry dropped.
func TestAgentsPruneOrphanAgent(t *testing.T) {
	home := t.TempDir()
	cfg, cfgDir := addWorkspace(t, copyAgentTOML, map[string]string{"rev": "# rev\n"})

	if out, err := runCmd(t, home, "", "agents", "add", "rev", "--config", cfg); err != nil {
		t.Fatalf("agents add: %v\n%s", err, out)
	}

	// De-declare rev: rewrite the same config (same dir => same lockfile) empty.
	if err := os.WriteFile(cfg, []byte("\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCmd(t, home, "", "agents", "prune", "--config", cfg)
	if err != nil {
		t.Fatalf("prune: %v\n%s", err, out)
	}

	// Install files gone from disk.
	for _, dst := range []string{claudeDst(home, "rev"), opencodeDst(home, "rev")} {
		if _, err := os.Lstat(dst); !os.IsNotExist(err) {
			t.Fatalf("orphan install file must be removed: %s (err %v)", dst, err)
		}
	}
	// Lockfile no longer records rev.
	lock, err := agentlock.Load(filepath.Join(cfgDir, ".homonto"))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := lock.Agents["rev"]; ok {
		t.Fatalf("orphan lockfile entry must be dropped, still present: %+v", lock.Agents)
	}
}

// TestAgentsPruneDeDeclaredTarget: a target no longer declared is removed and
// dropped from Installed, while the agent and its still-declared target remain.
func TestAgentsPruneDeDeclaredTarget(t *testing.T) {
	home := t.TempDir()
	cfg, cfgDir := addWorkspace(t, copyAgentTOML, map[string]string{"rev": "# rev\n"})

	if out, err := runCmd(t, home, "", "agents", "add", "rev", "--config", cfg); err != nil {
		t.Fatalf("agents add: %v\n%s", err, out)
	}

	// Re-declare rev targeting ONLY claude (drop opencode).
	claudeOnly := `
[agents.rev]
source = "local:rev"
mode = "copy"
targets = ["claude"]
`
	if err := os.WriteFile(cfg, []byte(claudeOnly), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCmd(t, home, "", "agents", "prune", "--config", cfg)
	if err != nil {
		t.Fatalf("prune: %v\n%s", err, out)
	}

	// opencode install removed, claude kept.
	if _, err := os.Lstat(opencodeDst(home, "rev")); !os.IsNotExist(err) {
		t.Fatalf("de-declared opencode target must be removed (err %v)", err)
	}
	if _, err := os.Lstat(claudeDst(home, "rev")); err != nil {
		t.Fatalf("still-declared claude target must be kept: %v", err)
	}

	lock, err := agentlock.Load(filepath.Join(cfgDir, ".homonto"))
	if err != nil {
		t.Fatal(err)
	}
	rec, ok := lock.Agents["rev"]
	if !ok {
		t.Fatalf("agent rev must remain in the lockfile")
	}
	if _, ok := rec.Installed["opencode"]; ok {
		t.Fatalf("opencode Installed entry must be dropped: %+v", rec.Installed)
	}
	if _, ok := rec.Installed["claude"]; !ok {
		t.Fatalf("claude Installed entry must remain: %+v", rec.Installed)
	}
}

// TestAgentsPruneBacksUpLocalEdit: an orphan install whose on-disk content was
// locally edited (differs from the recorded base hash) is backed up to <dst>.bak
// before removal.
func TestAgentsPruneBacksUpLocalEdit(t *testing.T) {
	home := t.TempDir()
	cfg, cfgDir := addWorkspace(t, copyAgentTOML, map[string]string{"rev": "# rev\n"})

	if out, err := runCmd(t, home, "", "agents", "add", "rev", "--config", cfg); err != nil {
		t.Fatalf("agents add: %v\n%s", err, out)
	}

	// Locally edit the claude install so it differs from the recorded base hash.
	dst := claudeDst(home, "rev")
	edited := "# rev LOCALLY EDITED\n"
	if err := os.WriteFile(dst, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}

	// Make rev an orphan.
	if err := os.WriteFile(cfg, []byte("\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if out, err := runCmd(t, home, "", "agents", "prune", "--config", cfg); err != nil {
		t.Fatalf("prune: %v\n%s", err, out)
	}

	// The edited content is preserved at <dst>.bak.
	bak, err := os.ReadFile(dst + ".bak")
	if err != nil {
		t.Fatalf("locally-edited file must be backed up to %s.bak: %v", dst, err)
	}
	if string(bak) != edited {
		t.Fatalf(".bak must hold the edited content, got:\n%s", bak)
	}
	// The install itself is removed.
	if _, err := os.Lstat(dst); !os.IsNotExist(err) {
		t.Fatalf("install must be removed after backup (err %v)", err)
	}
	// Lockfile entry dropped.
	lock, _ := agentlock.Load(filepath.Join(cfgDir, ".homonto"))
	if _, ok := lock.Agents["rev"]; ok {
		t.Fatalf("orphan entry must be dropped")
	}
}

// TestAgentsPruneBackupFailureKeepsFile: if the .bak backup of a locally-edited
// install cannot be written, the install file must NOT be removed (no data
// loss) and its lockfile record must be kept.
func TestAgentsPruneBackupFailureKeepsFile(t *testing.T) {
	home := t.TempDir()
	cfg, cfgDir := addWorkspace(t, copyAgentTOML, map[string]string{"rev": "# rev\n"})
	if out, err := runCmd(t, home, "", "agents", "add", "rev", "--config", cfg); err != nil {
		t.Fatalf("agents add: %v\n%s", err, out)
	}
	dst := claudeDst(home, "rev")
	edited := "# rev LOCALLY EDITED\n"
	if err := os.WriteFile(dst, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	// Force the backup write to fail: make <dst>.bak a non-empty directory so the
	// atomic rename onto it cannot succeed.
	if err := os.Mkdir(dst+".bak", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dst+".bak", "x"), []byte("y"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Make rev an orphan.
	if err := os.WriteFile(cfg, []byte("\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCmd(t, home, "", "agents", "prune", "--config", cfg)
	if err != nil {
		t.Fatalf("prune should not hard-error on a backup failure: %v\n%s", err, out)
	}
	if !strings.Contains(out, "SKIPPED") {
		t.Fatalf("prune must report the skipped file, got:\n%s", out)
	}
	// The edited install must still exist (not removed without a backup).
	if got, rerr := os.ReadFile(dst); rerr != nil || string(got) != edited {
		t.Fatalf("locally-edited install must be KEPT when backup fails: err=%v content=%q", rerr, got)
	}
	// The lockfile record must be kept (so a retry can prune it).
	lock, _ := agentlock.Load(filepath.Join(cfgDir, ".homonto"))
	if _, ok := lock.Agents["rev"]; !ok {
		t.Fatalf("record must be kept when the file could not be safely pruned")
	}
}

// TestAgentsPruneRemovesMergedSidecar: a leftover <dst>.merged next to a pruned
// orphan target is removed too.
func TestAgentsPruneRemovesMergedSidecar(t *testing.T) {
	home := t.TempDir()
	cfg, _ := addWorkspace(t, copyAgentTOML, map[string]string{"rev": "# rev\n"})

	if out, err := runCmd(t, home, "", "agents", "add", "rev", "--config", cfg); err != nil {
		t.Fatalf("agents add: %v\n%s", err, out)
	}

	dst := claudeDst(home, "rev")
	if err := os.WriteFile(dst+".merged", []byte("<<<< conflict\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Make rev an orphan.
	if err := os.WriteFile(cfg, []byte("\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if out, err := runCmd(t, home, "", "agents", "prune", "--config", cfg); err != nil {
		t.Fatalf("prune: %v\n%s", err, out)
	}

	if _, err := os.Lstat(dst + ".merged"); !os.IsNotExist(err) {
		t.Fatalf("the .merged sidecar must be removed (err %v)", err)
	}
}

// TestAgentsPruneNothingToPrune: a lockfile that matches the config exactly
// yields "nothing to prune" and changes nothing.
func TestAgentsPruneNothingToPrune(t *testing.T) {
	home := t.TempDir()
	cfg, cfgDir := addWorkspace(t, copyAgentTOML, map[string]string{"rev": "# rev\n"})

	if out, err := runCmd(t, home, "", "agents", "add", "rev", "--config", cfg); err != nil {
		t.Fatalf("agents add: %v\n%s", err, out)
	}

	lockPath := filepath.Join(cfgDir, ".homonto", "agents-lock.json")
	before, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}

	out, err := runCmd(t, home, "", "agents", "prune", "--config", cfg)
	if err != nil {
		t.Fatalf("prune: %v\n%s", err, out)
	}
	if !strings.Contains(out, "nothing to prune") {
		t.Fatalf("must report nothing to prune, got:\n%s", out)
	}

	after, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatalf("lockfile must be unchanged:\nbefore %s\nafter %s", before, after)
	}
	for _, dst := range []string{claudeDst(home, "rev"), opencodeDst(home, "rev")} {
		if _, err := os.Lstat(dst); err != nil {
			t.Fatalf("install file must remain intact: %s (%v)", dst, err)
		}
	}
}

// TestAgentsPruneDryRun: --dry-run lists what would be pruned but removes no
// file and does not mutate the lockfile.
func TestAgentsPruneDryRun(t *testing.T) {
	home := t.TempDir()
	cfg, cfgDir := addWorkspace(t, copyAgentTOML, map[string]string{"rev": "# rev\n"})

	if out, err := runCmd(t, home, "", "agents", "add", "rev", "--config", cfg); err != nil {
		t.Fatalf("agents add: %v\n%s", err, out)
	}

	// Make rev an orphan.
	if err := os.WriteFile(cfg, []byte("\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	lockPath := filepath.Join(cfgDir, ".homonto", "agents-lock.json")
	before, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}

	out, err := runCmd(t, home, "", "agents", "prune", "--dry-run", "--config", cfg)
	if err != nil {
		t.Fatalf("prune --dry-run: %v\n%s", err, out)
	}
	if !strings.Contains(out, "would remove") {
		t.Fatalf("dry run must list what would be removed, got:\n%s", out)
	}

	// No file removed.
	for _, dst := range []string{claudeDst(home, "rev"), opencodeDst(home, "rev")} {
		if _, err := os.Lstat(dst); err != nil {
			t.Fatalf("dry run must not remove install file %s: %v", dst, err)
		}
	}
	// Lockfile unchanged and still records rev.
	after, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatalf("dry run must not mutate the lockfile:\nbefore %s\nafter %s", before, after)
	}
	lock, _ := agentlock.Load(filepath.Join(cfgDir, ".homonto"))
	if _, ok := lock.Agents["rev"]; !ok {
		t.Fatalf("dry run must keep the lockfile entry")
	}
}

// TestAgentsPruneDeletionFailureKeepsRecord: if os.Remove of the install cannot
// succeed, prune must NOT report it "removed" or drop the lockfile record.
// Ownership is retained so a later prune can retry. Complements the existing
// backup-failure test, which only covers a failed .bak write.
func TestAgentsPruneDeletionFailureKeepsRecord(t *testing.T) {
	home := t.TempDir()
	cfg, cfgDir := addWorkspace(t, copyAgentTOML, map[string]string{"rev": "# rev\n"})
	if out, err := runCmd(t, home, "", "agents", "add", "rev", "--config", cfg); err != nil {
		t.Fatalf("agents add: %v\n%s", err, out)
	}
	dst := claudeDst(home, "rev")

	// Force os.Remove(dst) to fail deterministically, even under root: replace the
	// install file with a NON-EMPTY directory (rmdir refuses with ENOTEMPTY).
	if err := os.Remove(dst); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dst, "blocker"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Make rev an orphan so prune tries to remove it.
	if err := os.WriteFile(cfg, []byte("\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCmd(t, home, "", "agents", "prune", "--config", cfg)
	if err != nil {
		t.Fatalf("prune should not hard-error on a deletion failure: %v\n%s", err, out)
	}
	if strings.Contains(out, "removed "+dst) {
		t.Fatalf("prune must not report a failed deletion as removed:\n%s", out)
	}
	if !strings.Contains(out, "SKIPPED") {
		t.Fatalf("prune must report the skipped file, got:\n%s", out)
	}
	// The lockfile record must be kept (ownership retained for a retry).
	lock, _ := agentlock.Load(filepath.Join(cfgDir, ".homonto"))
	if _, ok := lock.Agents["rev"]; !ok {
		t.Fatalf("record must be kept when the file could not be removed")
	}
}
