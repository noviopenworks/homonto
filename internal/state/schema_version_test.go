package state

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSave_StampsSchemaVersion(t *testing.T) {
	dir := t.TempDir()
	if err := newState().Save(dir); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.SchemaVersion != CurrentStateSchemaVersion {
		t.Errorf("SchemaVersion = %d, want %d", loaded.SchemaVersion, CurrentStateSchemaVersion)
	}
}

func TestLoad_RejectsFutureSchemaVersion(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "state.json"), []byte(`{"schemaVersion":999,"managed":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(dir)
	if err == nil {
		t.Fatal("future schemaVersion accepted, want rejection")
	}
	if !strings.Contains(err.Error(), "schema") {
		t.Errorf("error %q should mention the schema version", err.Error())
	}
}

func TestLoad_LegacyNoVersionStillLoads(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "state.json"), []byte(`{"managed":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err != nil {
		t.Fatalf("legacy state (no schemaVersion) rejected: %v", err)
	}
}
