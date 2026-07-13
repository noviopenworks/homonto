package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestImportForce_BacksUpExistingConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := filepath.Join(t.TempDir(), "homonto.toml")
	const old = "# my hand-tuned config\n[mcps.keep]\ncommand = [\"true\"]\n"
	if err := os.WriteFile(cfg, []byte(old), 0o644); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"import", "--force", "--config", cfg})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("import --force: %v (out=%q)", err, out.String())
	}

	bak, err := os.ReadFile(cfg + ".bak")
	if err != nil {
		t.Fatalf("expected backup at %s.bak: %v", cfg, err)
	}
	if string(bak) != old {
		t.Errorf("backup content = %q, want the old config %q", string(bak), old)
	}
	// the new config was written (may differ from old); just assert it exists
	if _, err := os.Stat(cfg); err != nil {
		t.Errorf("new config missing after import: %v", err)
	}
}
