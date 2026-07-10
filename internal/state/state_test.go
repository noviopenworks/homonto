package state

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAbsentReturnsEmpty(t *testing.T) {
	s, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if _, ok := s.Get("claude", "setting.model"); ok {
		t.Fatal("expected empty state")
	}
}

func TestSaveAndReloadEntry(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested")
	s, _ := Load(dir)
	s.Set("claude", "setting.model", `"opus"`, "abc123hash")
	if err := s.Save(dir); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, _ := Load(dir)
	e, ok := got.Get("claude", "setting.model")
	if !ok {
		t.Fatal("entry missing after reload")
	}
	if e.Desired != `"opus"` || e.Applied != "abc123hash" {
		t.Fatalf("reloaded entry = %+v", e)
	}
}

func TestKeysSortedAndDelete(t *testing.T) {
	s, _ := Load(t.TempDir())
	s.Set("claude", "mcp.b", "x", "h")
	s.Set("claude", "mcp.a", "x", "h")
	s.Set("opencode", "mcp.c", "x", "h")
	got := s.Keys("claude")
	if len(got) != 2 || got[0] != "mcp.a" || got[1] != "mcp.b" {
		t.Fatalf("Keys = %v, want sorted [mcp.a mcp.b]", got)
	}
	if len(s.Keys("unknown-tool")) != 0 {
		t.Fatal("Keys for an unknown tool must be empty")
	}
	s.Delete("claude", "mcp.a")
	if _, ok := s.Get("claude", "mcp.a"); ok {
		t.Fatal("entry survives Delete")
	}
	if _, ok := s.Get("claude", "mcp.b"); !ok {
		t.Fatal("Delete removed the wrong entry")
	}
}

func TestCatalogVersionRoundTrips(t *testing.T) {
	dir := t.TempDir()
	s, _ := Load(dir)
	if s.CatalogVersionRecorded() != "" {
		t.Fatal("fresh state should record no catalog version")
	}
	s.SetCatalogVersion("0.1.0")
	if err := s.Save(dir); err != nil {
		t.Fatal(err)
	}
	got, _ := Load(dir)
	if got.CatalogVersionRecorded() != "0.1.0" {
		t.Fatalf("reloaded catalog version = %q", got.CatalogVersionRecorded())
	}
}

func TestCatalogVersionOmittedWhenEmpty(t *testing.T) {
	dir := t.TempDir()
	s, _ := Load(dir)
	s.Set("claude", "mcp.a", "x", "h") // some content so the file is written
	if err := s.Save(dir); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(dir, "state.json"))
	if strings.Contains(string(raw), "catalogVersion") {
		t.Fatalf("empty catalog version must be omitted, got %s", raw)
	}
}

func TestSaveIsAtomicJSON(t *testing.T) {
	dir := t.TempDir()
	s, _ := Load(dir)
	s.Set("opencode", "mcp.brave", `{"env":{"K":"${pass:x}"}}`, "deadbeef")
	if err := s.Save(dir); err != nil {
		t.Fatal(err)
	}
	// The unresolved token is stored under desired; a hash under applied.
	raw, _ := os.ReadFile(filepath.Join(dir, "state.json"))
	if !strings.Contains(string(raw), "${pass:x}") || !strings.Contains(string(raw), "deadbeef") {
		t.Fatalf("state.json = %s", raw)
	}
}
