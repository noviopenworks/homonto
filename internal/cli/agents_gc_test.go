package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/noviopenworks/homonto/internal/agentlock"
)

// TestAgentsGCReclaimsUnreferencedBlobs: `agents update` advancing a base leaves
// the superseded base blob unreferenced; `agents gc` reclaims it (and only it),
// keeping the live base. --dry-run previews without removing.
func TestAgentsGCReclaimsUnreferencedBlobs(t *testing.T) {
	home := t.TempDir()
	cfg, cfgDir := addWorkspace(t, copyAgentTOML, map[string]string{"rev": "# rev v1\n"})
	if out, err := runCmd(t, home, "", "agents", "add", "rev", "--config", cfg); err != nil {
		t.Fatalf("add: %v\n%s", err, out)
	}
	blobs := filepath.Join(cfgDir, ".homonto", "agents-blobs")

	// Change the source and update → clean overwrite advances the recorded base
	// to v2; the v1 blob is now dead.
	srcPath := filepath.Join(cfgDir, "homonto", "agents", "rev.md")
	if err := os.WriteFile(srcPath, []byte("# rev v2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := runCmd(t, home, "", "agents", "update", "rev", "--config", cfg); err != nil {
		t.Fatalf("update: %v\n%s", err, out)
	}
	if ents, _ := os.ReadDir(blobs); len(ents) != 2 {
		t.Fatalf("expected 2 blobs after update (v1 dead + v2 live), got %d", len(ents))
	}

	// Dry-run previews and removes nothing.
	out, err := runCmd(t, home, "", "agents", "gc", "--dry-run", "--config", cfg)
	if err != nil {
		t.Fatalf("gc --dry-run: %v\n%s", err, out)
	}
	if !strings.Contains(out, "would reclaim") {
		t.Fatalf("dry-run must preview a reclaim, got:\n%s", out)
	}
	if ents, _ := os.ReadDir(blobs); len(ents) != 2 {
		t.Fatalf("dry-run must not remove blobs, got %d", len(ents))
	}

	// Real gc removes the dead v1 blob and keeps the live v2 base.
	if out, err := runCmd(t, home, "", "agents", "gc", "--config", cfg); err != nil {
		t.Fatalf("gc: %v\n%s", err, out)
	}
	ents, _ := os.ReadDir(blobs)
	if len(ents) != 1 {
		t.Fatalf("gc should leave exactly the 1 live blob, got %d", len(ents))
	}
	lock, _ := agentlock.Load(filepath.Join(cfgDir, ".homonto"))
	liveHash := ""
	for _, ins := range lock.Agents["rev"].Installed {
		liveHash = ins.Hash
	}
	if ents[0].Name() != liveHash {
		t.Fatalf("surviving blob %s must be the live base hash %s", ents[0].Name(), liveHash)
	}

	// A second gc is a no-op.
	out, err = runCmd(t, home, "", "agents", "gc", "--config", cfg)
	if err != nil {
		t.Fatalf("gc (2nd): %v\n%s", err, out)
	}
	if !strings.Contains(out, "no unreferenced blobs") {
		t.Fatalf("second gc must be a no-op, got:\n%s", out)
	}
}
