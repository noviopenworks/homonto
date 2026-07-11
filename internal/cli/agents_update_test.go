package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/noviopenworks/homonto/internal/agentlock"
	"github.com/noviopenworks/homonto/internal/subagentpath"
)

// TestAgentsUpdateSourceChangedInstallUntouched: when the provider source changes
// but the installed copies were not touched, update re-materializes the new
// content to every target, refreshes the lockfile hash, and creates NO .bak
// (an untouched install of an older source is not a local edit).
func TestAgentsUpdateSourceChangedInstallUntouched(t *testing.T) {
	home := t.TempDir()
	old := "# Rev agent v1\n"
	cfg, cfgDir := addWorkspace(t, copyAgentTOML, map[string]string{"rev": old})

	if out, err := runCmd(t, home, "", "agents", "add", "rev", "--config", cfg); err != nil {
		t.Fatalf("agents add: %v\n%s", err, out)
	}

	// Change the provider source; leave installed copies untouched.
	neu := "# Rev agent v2 (new source)\n"
	srcPath := filepath.Join(cfgDir, "homonto", "agents", "rev.md")
	if err := os.WriteFile(srcPath, []byte(neu), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCmd(t, home, "", "agents", "update", "rev", "--config", cfg)
	if err != nil {
		t.Fatalf("update: %v\n%s", err, out)
	}

	wantHash := agentlock.HashContent([]byte(neu))
	for _, tool := range []string{"claude", "opencode"} {
		dst := filepath.Join(subagentpath.Dir(tool, "user", home, ""), "rev.md")
		got, rerr := os.ReadFile(dst)
		if rerr != nil {
			t.Fatalf("%s: expected file at %s: %v", tool, dst, rerr)
		}
		if string(got) != neu {
			t.Fatalf("%s: expected refreshed content, got:\n%s", tool, got)
		}
		if _, err := os.Stat(dst + ".bak"); !os.IsNotExist(err) {
			t.Fatalf("%s: untouched install must NOT be backed up, but %s.bak exists", tool, dst)
		}
	}

	lock, err := agentlock.Load(filepath.Join(cfgDir, ".homonto"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range []string{"claude", "opencode"} {
		if ins := lock.Agents["rev"].Installed[tool]; ins.Hash != wantHash {
			t.Fatalf("%s: lockfile hash not refreshed: got %s want %s", tool, ins.Hash, wantHash)
		}
	}
}

// TestAgentsUpdateBacksUpLocalEdit: on a CLEAN three-way merge (disjoint local
// and source edits) the merged result is written to <dst> and the pre-merge
// local is preserved in <dst>.bak.
func TestAgentsUpdateBacksUpLocalEdit(t *testing.T) {
	home := t.TempDir()
	old := "alpha\nbeta\ngamma\n"
	cfg, cfgDir := addWorkspace(t, copyAgentTOML, map[string]string{"rev": old})

	if out, err := runCmd(t, home, "", "agents", "add", "rev", "--config", cfg); err != nil {
		t.Fatalf("agents add: %v\n%s", err, out)
	}

	// Locally edit line 1 of the claude copy (disjoint from the source edit).
	localX := "ALPHA-LOCAL\nbeta\ngamma\n"
	claudeDst := filepath.Join(subagentpath.Dir("claude", "user", home, ""), "rev.md")
	if err := os.WriteFile(claudeDst, []byte(localX), 0o644); err != nil {
		t.Fatal(err)
	}
	// Change source line 3 (disjoint from the local edit) → clean merge.
	srcY := "alpha\nbeta\nGAMMA-SOURCE\n"
	srcPath := filepath.Join(cfgDir, "homonto", "agents", "rev.md")
	if err := os.WriteFile(srcPath, []byte(srcY), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCmd(t, home, "", "agents", "update", "rev", "--config", cfg)
	if err != nil {
		t.Fatalf("update: %v\n%s", err, out)
	}

	got, _ := os.ReadFile(claudeDst)
	if !strings.Contains(string(got), "ALPHA-LOCAL") || !strings.Contains(string(got), "GAMMA-SOURCE") {
		t.Fatalf("claude dst must hold the merged result with both edits, got:\n%s", got)
	}
	bak, berr := os.ReadFile(claudeDst + ".bak")
	if berr != nil {
		t.Fatalf("pre-merge local must be backed up to %s.bak: %v", claudeDst, berr)
	}
	if string(bak) != localX {
		t.Fatalf("%s.bak must preserve the pre-merge local X, got:\n%s", claudeDst, bak)
	}
}

// TestAgentsUpdateDisjointEditsAutoMerge: a local edit and a disjoint source edit
// three-way merge cleanly — <dst> ends up with BOTH edits, no <dst>.merged is
// created, and the recorded base advances to the source (doctor is healthy after).
func TestAgentsUpdateDisjointEditsAutoMerge(t *testing.T) {
	home := t.TempDir()
	base := "line1\nline2\nline3\nline4\nline5\nline6\n"
	toml := "[agents.rev]\nsource=\"local:rev\"\nmode=\"copy\"\ntargets=[\"claude\"]\n"
	cfg, cfgDir := addWorkspace(t, toml, map[string]string{"rev": base})

	if out, err := runCmd(t, home, "", "agents", "add", "rev", "--config", cfg); err != nil {
		t.Fatalf("agents add: %v\n%s", err, out)
	}

	dst := filepath.Join(subagentpath.Dir("claude", "user", home, ""), "rev.md")
	localEdit := "LOCALEDIT1\nline2\nline3\nline4\nline5\nline6\n"
	if err := os.WriteFile(dst, []byte(localEdit), 0o644); err != nil {
		t.Fatal(err)
	}
	newSrc := "line1\nline2\nline3\nline4\nline5\nline6-UPSTREAM\n"
	srcPath := filepath.Join(cfgDir, "homonto", "agents", "rev.md")
	if err := os.WriteFile(srcPath, []byte(newSrc), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCmd(t, home, "", "agents", "update", "rev", "--config", cfg)
	if err != nil {
		t.Fatalf("clean auto-merge must succeed, got %v\n%s", err, out)
	}

	got, _ := os.ReadFile(dst)
	if !strings.Contains(string(got), "LOCALEDIT1") || !strings.Contains(string(got), "line6-UPSTREAM") {
		t.Fatalf("merged dst must contain BOTH edits, got:\n%s", got)
	}
	if _, err := os.Stat(dst + ".merged"); !os.IsNotExist(err) {
		t.Fatalf("clean auto-merge must NOT create %s.merged", dst)
	}

	// Base advanced to the new source → doctor is healthy (exit 0, no findings).
	dout, derr := runCmd(t, home, "", "agents", "doctor", "--config", cfg)
	if derr != nil {
		t.Fatalf("doctor after clean merge must be healthy, got %v\n%s", derr, dout)
	}
	if dout != "healthy\n" {
		t.Fatalf("doctor after clean merge must print exactly \"healthy\", got:\n%q", dout)
	}
}

// TestAgentsUpdateOverlappingConflictSidecar: when the local edit and the source
// edit overlap, the live <dst> is left UNCHANGED, a <dst>.merged sidecar with
// conflict markers is written, the lockfile entry stays at the prior base, and
// the command exits non-zero.
func TestAgentsUpdateOverlappingConflictSidecar(t *testing.T) {
	home := t.TempDir()
	base := "line1\nline2\nline3\nline4\n"
	toml := "[agents.rev]\nsource=\"local:rev\"\nmode=\"copy\"\ntargets=[\"claude\"]\n"
	cfg, cfgDir := addWorkspace(t, toml, map[string]string{"rev": base})

	if out, err := runCmd(t, home, "", "agents", "add", "rev", "--config", cfg); err != nil {
		t.Fatalf("agents add: %v\n%s", err, out)
	}

	dst := filepath.Join(subagentpath.Dir("claude", "user", home, ""), "rev.md")
	localEdit := "line1\nline2\nLOCAL3\nline4\n"
	if err := os.WriteFile(dst, []byte(localEdit), 0o644); err != nil {
		t.Fatal(err)
	}
	newSrc := "line1\nline2\nUPSTREAM3\nline4\n"
	srcPath := filepath.Join(cfgDir, "homonto", "agents", "rev.md")
	if err := os.WriteFile(srcPath, []byte(newSrc), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCmd(t, home, "", "agents", "update", "rev", "--config", cfg)
	if err == nil {
		t.Fatalf("overlapping conflict must exit non-zero, got:\n%s", out)
	}

	// Live dst is untouched — still exactly the local edit.
	if got, _ := os.ReadFile(dst); string(got) != localEdit {
		t.Fatalf("live dst must be UNCHANGED on conflict, got:\n%s", got)
	}
	// Sidecar exists with git-style conflict markers.
	m, merr := os.ReadFile(dst + ".merged")
	if merr != nil {
		t.Fatalf(".merged sidecar must be written on conflict: %v", merr)
	}
	if !strings.Contains(string(m), "<<<<<<<") || !strings.Contains(string(m), ">>>>>>>") {
		t.Fatalf("%s.merged must contain conflict markers, got:\n%s", dst, m)
	}
	// Lockfile entry for the conflicted target stays at the prior base hash.
	lock, err := agentlock.Load(filepath.Join(cfgDir, ".homonto"))
	if err != nil {
		t.Fatal(err)
	}
	if got := lock.Agents["rev"].Installed["claude"].Hash; got != agentlock.HashContent([]byte(base)) {
		t.Fatalf("conflicted target lockfile hash must stay prev (base), got %s", got)
	}
}

// TestAgentsUpdateMissingBaseFallsBackToBackup: when the base blob is gone, a
// three-way merge is impossible, so update falls back to backup-before-overwrite —
// the local edit lands in <dst>.bak and the source overwrites <dst>.
func TestAgentsUpdateMissingBaseFallsBackToBackup(t *testing.T) {
	home := t.TempDir()
	base := "# v1\n"
	toml := "[agents.rev]\nsource=\"local:rev\"\nmode=\"copy\"\ntargets=[\"claude\"]\n"
	cfg, cfgDir := addWorkspace(t, toml, map[string]string{"rev": base})

	if out, err := runCmd(t, home, "", "agents", "add", "rev", "--config", cfg); err != nil {
		t.Fatalf("agents add: %v\n%s", err, out)
	}

	// Delete the recorded base blob so no ancestor is available.
	homontoDir := filepath.Join(cfgDir, ".homonto")
	baseHash := agentlock.HashContent([]byte(base))
	if err := os.Remove(filepath.Join(homontoDir, "agents-blobs", baseHash)); err != nil {
		t.Fatal(err)
	}

	dst := filepath.Join(subagentpath.Dir("claude", "user", home, ""), "rev.md")
	localEdit := "# local edit\n"
	if err := os.WriteFile(dst, []byte(localEdit), 0o644); err != nil {
		t.Fatal(err)
	}
	newSrc := "# v2\n"
	srcPath := filepath.Join(cfgDir, "homonto", "agents", "rev.md")
	if err := os.WriteFile(srcPath, []byte(newSrc), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCmd(t, home, "", "agents", "update", "rev", "--config", cfg)
	if err != nil {
		t.Fatalf("missing-base fallback must succeed, got %v\n%s", err, out)
	}
	if got, _ := os.ReadFile(dst); string(got) != newSrc {
		t.Fatalf("fallback must overwrite dst with the source, got:\n%s", got)
	}
	bak, berr := os.ReadFile(dst + ".bak")
	if berr != nil {
		t.Fatalf("fallback must back up the local edit to %s.bak: %v", dst, berr)
	}
	if string(bak) != localEdit {
		t.Fatalf("%s.bak must preserve the local edit, got:\n%s", dst, bak)
	}
	if _, err := os.Stat(dst + ".merged"); !os.IsNotExist(err) {
		t.Fatalf("fallback must NOT create %s.merged", dst)
	}
}

// TestAgentsUpdateNewTargetBacksUpForeignFile: when a target is newly added to
// the config after install, update must NOT silently clobber a pre-existing
// foreign file at that target — it backs it up first.
func TestAgentsUpdateNewTargetBacksUpForeignFile(t *testing.T) {
	home := t.TempDir()
	claudeOnly := "[agents.rev]\nsource = \"local:rev\"\nmode = \"copy\"\ntargets = [\"claude\"]\n"
	cfg, cfgDir := addWorkspace(t, claudeOnly, map[string]string{"rev": "# rev v1\n"})

	if out, err := runCmd(t, home, "", "agents", "add", "rev", "--config", cfg); err != nil {
		t.Fatalf("agents add: %v\n%s", err, out)
	}

	// A hand-written foreign file already sits at the opencode target path.
	foreign := "# hand-written, do not lose\n"
	ocDst := filepath.Join(subagentpath.Dir("opencode", "user", home, ""), "rev.md")
	if err := os.MkdirAll(filepath.Dir(ocDst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ocDst, []byte(foreign), 0o644); err != nil {
		t.Fatal(err)
	}

	// Now the config adds opencode as a target.
	both := "[agents.rev]\nsource = \"local:rev\"\nmode = \"copy\"\ntargets = [\"claude\", \"opencode\"]\n"
	if err := os.WriteFile(cfg, []byte(both), 0o644); err != nil {
		t.Fatal(err)
	}

	if out, err := runCmd(t, home, "", "agents", "update", "rev", "--config", cfg); err != nil {
		t.Fatalf("update: %v\n%s", err, out)
	}
	// The foreign file must have been backed up, not silently destroyed.
	bak, berr := os.ReadFile(ocDst + ".bak")
	if berr != nil {
		t.Fatalf("foreign file at a new target must be backed up to %s.bak: %v", ocDst, berr)
	}
	if string(bak) != foreign {
		t.Fatalf("%s.bak must preserve the foreign content, got:\n%s", ocDst, bak)
	}
	if got, _ := os.ReadFile(ocDst); string(got) != "# rev v1\n" {
		t.Fatalf("opencode dst should hold the source content, got:\n%s", got)
	}
	_ = cfgDir
}

// TestAgentsUpdateIsIdempotent: updating immediately after add reports every
// target "up to date", writes no .bak, and leaves dst bytes unchanged.
func TestAgentsUpdateIsIdempotent(t *testing.T) {
	home := t.TempDir()
	body := "# Rev agent\n"
	cfg, _ := addWorkspace(t, copyAgentTOML, map[string]string{"rev": body})

	if out, err := runCmd(t, home, "", "agents", "add", "rev", "--config", cfg); err != nil {
		t.Fatalf("agents add: %v\n%s", err, out)
	}

	claudeDst := filepath.Join(subagentpath.Dir("claude", "user", home, ""), "rev.md")
	fiBefore, err := os.Stat(claudeDst)
	if err != nil {
		t.Fatal(err)
	}

	out, err := runCmd(t, home, "", "agents", "update", "rev", "--config", cfg)
	if err != nil {
		t.Fatalf("update: %v\n%s", err, out)
	}
	if !strings.Contains(out, "up to date") {
		t.Fatalf("idempotent update must report up to date, got:\n%s", out)
	}
	if _, err := os.Stat(claudeDst + ".bak"); !os.IsNotExist(err) {
		t.Fatalf("idempotent update must NOT create %s.bak", claudeDst)
	}
	if _, err := os.Stat(claudeDst + ".merged"); !os.IsNotExist(err) {
		t.Fatalf("idempotent update must NOT create %s.merged", claudeDst)
	}
	fiAfter, err := os.Stat(claudeDst)
	if err != nil {
		t.Fatal(err)
	}
	if !fiAfter.ModTime().Equal(fiBefore.ModTime()) {
		t.Fatalf("idempotent update must not rewrite dst (mtime changed): %v -> %v", fiBefore.ModTime(), fiAfter.ModTime())
	}
	if got, _ := os.ReadFile(claudeDst); string(got) != body {
		t.Fatalf("content changed on idempotent update:\n%s", got)
	}
}

// TestAgentsUpdateNotInstalled: updating a declared-but-not-installed agent errors,
// pointing at `agents add`.
func TestAgentsUpdateNotInstalled(t *testing.T) {
	home := t.TempDir()
	toml := "[agents.new]\nsource=\"local:new\"\nmode=\"copy\"\n"
	cfg, _ := addWorkspace(t, toml, map[string]string{"new": "# new\n"})

	out, err := runCmd(t, home, "", "agents", "update", "new", "--config", cfg)
	if err == nil {
		t.Fatalf("update of a not-installed agent must error, got:\n%s", out)
	}
	if !strings.Contains(err.Error(), "not installed") || !strings.Contains(err.Error(), "agents add") {
		t.Fatalf("error must say not installed and point at `agents add`, got: %v", err)
	}
}

// TestAgentsUpdateBuiltinLinkIsError: builtin + link is rejected on update too
// (builtin has no local path to symlink).
func TestAgentsUpdateBuiltinLinkIsError(t *testing.T) {
	home := t.TempDir()
	cfg, _ := addWorkspace(t, "[agents.cr]\nsource=\"builtin:code-reviewer\"\nmode=\"link\"\ntargets=[\"claude\"]\n", nil)
	out, err := runCmd(t, home, "", "agents", "update", "cr", "--config", cfg)
	if err == nil {
		t.Fatalf("builtin + link update must be refused, got:\n%s", out)
	}
	if !strings.Contains(err.Error(), "link") || !strings.Contains(err.Error(), "builtin") {
		t.Fatalf("error must explain builtin cannot use link mode, got: %v", err)
	}
}

// TestAgentsUpdateUndeclared: updating an undeclared agent errors clearly.
func TestAgentsUpdateUndeclared(t *testing.T) {
	home := t.TempDir()
	cfg, _ := addWorkspace(t, copyAgentTOML, map[string]string{"rev": "x"})
	out, err := runCmd(t, home, "", "agents", "update", "nope", "--config", cfg)
	if err == nil {
		t.Fatalf("undeclared agent must error, got:\n%s", out)
	}
	if !strings.Contains(err.Error(), "not declared") {
		t.Fatalf("error must say not declared, got: %v", err)
	}
}

// TestAgentsUpdateLinkModeUpToDate: a link-mode agent whose symlink already points
// at the source is reported up to date and stays a symlink to the source.
func TestAgentsUpdateLinkModeUpToDate(t *testing.T) {
	home := t.TempDir()
	toml := `
[agents.rev]
source = "local:rev"
mode = "link"
targets = ["claude"]
`
	cfg, cfgDir := addWorkspace(t, toml, map[string]string{"rev": "# rev\n"})

	if out, err := runCmd(t, home, "", "agents", "add", "rev", "--config", cfg); err != nil {
		t.Fatalf("agents add: %v\n%s", err, out)
	}

	out, err := runCmd(t, home, "", "agents", "update", "rev", "--config", cfg)
	if err != nil {
		t.Fatalf("update: %v\n%s", err, out)
	}
	if !strings.Contains(out, "up to date") {
		t.Fatalf("link update must report up to date, got:\n%s", out)
	}

	dst := filepath.Join(subagentpath.Dir("claude", "user", home, ""), "rev.md")
	fi, err := os.Lstat(dst)
	if err != nil {
		t.Fatalf("link dst missing: %v", err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("link mode dst must remain a symlink at %s", dst)
	}
	target, err := os.Readlink(dst)
	if err != nil {
		t.Fatal(err)
	}
	srcPath := filepath.Join(cfgDir, "homonto", "agents", "rev.md")
	if target != srcPath {
		t.Fatalf("symlink must point at source %s, got %s", srcPath, target)
	}
}

// TestAgentsUpdateKeepsDeDeclaredTargetRecord: removing a target from the config
// then running `agents update` must NOT drop the prior install's lockfile record
// while its file remains on disk. Otherwise Homonto forgets it owns the file and
// a later prune cannot reclaim it. The record is kept (flagged for prune), and a
// subsequent prune then removes both the file and the record.
func TestAgentsUpdateKeepsDeDeclaredTargetRecord(t *testing.T) {
	home := t.TempDir()
	both := "[agents.rev]\nsource = \"local:rev\"\nmode = \"copy\"\ntargets = [\"claude\", \"opencode\"]\n"
	cfg, cfgDir := addWorkspace(t, both, map[string]string{"rev": "# rev v1\n"})

	if out, err := runCmd(t, home, "", "agents", "add", "rev", "--config", cfg); err != nil {
		t.Fatalf("agents add: %v\n%s", err, out)
	}
	ocDst := opencodeDst(home, "rev")
	if _, err := os.Stat(ocDst); err != nil {
		t.Fatalf("opencode install should exist after add: %v", err)
	}

	// De-declare opencode: claude only.
	claudeOnly := "[agents.rev]\nsource = \"local:rev\"\nmode = \"copy\"\ntargets = [\"claude\"]\n"
	if err := os.WriteFile(cfg, []byte(claudeOnly), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := runCmd(t, home, "", "agents", "update", "rev", "--config", cfg); err != nil {
		t.Fatalf("update: %v\n%s", err, out)
	}

	// The opencode file is still on disk, so ownership must NOT be forgotten.
	if _, err := os.Stat(ocDst); err != nil {
		t.Fatalf("de-declared opencode file should still be on disk: %v", err)
	}
	lock, err := agentlock.Load(filepath.Join(cfgDir, ".homonto"))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := lock.Agents["rev"].Installed["opencode"]; !ok {
		t.Fatalf("update dropped the de-declared opencode record while the file remains on disk (ownership lost)")
	}

	// Proof it is flagged for prune: prune now reclaims the file and the record.
	if out, err := runCmd(t, home, "", "agents", "prune", "--config", cfg); err != nil {
		t.Fatalf("prune: %v\n%s", err, out)
	}
	if _, err := os.Stat(ocDst); !os.IsNotExist(err) {
		t.Fatalf("prune should have removed the de-declared opencode file")
	}
	lock2, _ := agentlock.Load(filepath.Join(cfgDir, ".homonto"))
	if _, ok := lock2.Agents["rev"].Installed["opencode"]; ok {
		t.Fatalf("prune should have dropped the opencode record after removing the file")
	}
}
