package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "homonto.toml")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoad_RejectsFutureSchemaVersion(t *testing.T) {
	p := writeConfig(t, "schema_version = 999\n[mcps.demo]\ncommand = [\"true\"]\n")
	_, err := Load(p)
	if err == nil {
		t.Fatal("Load of a future schema_version should error")
	}
	if !strings.Contains(err.Error(), "upgrade homonto") {
		t.Errorf("error = %q, want an 'upgrade homonto' message", err)
	}
}

func TestLoad_AcceptsAbsentAndCurrentSchemaVersion(t *testing.T) {
	// Absent (legacy) loads fine.
	if _, err := Load(writeConfig(t, "[mcps.demo]\ncommand = [\"true\"]\n")); err != nil {
		t.Errorf("absent schema_version should load: %v", err)
	}
	// Explicit current version loads fine.
	body := "schema_version = 1\n[mcps.demo]\ncommand = [\"true\"]\n"
	if _, err := Load(writeConfig(t, body)); err != nil {
		t.Errorf("current schema_version should load: %v", err)
	}
}
