package catalog

import (
	"strings"
	"testing"
	"testing/fstest"
)

func manifestFS(manifest string) fstest.MapFS {
	return fstest.MapFS{
		"version.txt":                        {Data: []byte("0.1.0")},
		"frameworks/fw/framework.toml":       {Data: []byte(manifest)},
		"frameworks/fw/skills/demo/SKILL.md": {Data: []byte("demo")},
	}
}

func TestLoad_RejectsFutureManifestSchema(t *testing.T) {
	m := "manifest_schema = 999\nname = \"fw\"\nversion = \"0.1.0\"\n[skills]\ndemo = \"frameworks/fw/skills/demo\"\n"
	_, err := Load(manifestFS(m))
	if err == nil {
		t.Fatal("Load of a future manifest_schema should error")
	}
	if !strings.Contains(err.Error(), "upgrade homonto") {
		t.Errorf("error = %q, want an 'upgrade homonto' message", err)
	}
}

func TestLoad_AcceptsAbsentAndCurrentManifestSchema(t *testing.T) {
	// Absent (legacy) manifest loads.
	m0 := "name = \"fw\"\nversion = \"0.1.0\"\n[skills]\ndemo = \"frameworks/fw/skills/demo\"\n"
	if _, err := Load(manifestFS(m0)); err != nil {
		t.Errorf("absent manifest_schema should load: %v", err)
	}
	// Explicit current schema (1) loads.
	m1 := "manifest_schema = 1\nname = \"fw\"\nversion = \"0.1.0\"\n[skills]\ndemo = \"frameworks/fw/skills/demo\"\n"
	if _, err := Load(manifestFS(m1)); err != nil {
		t.Errorf("current manifest_schema should load: %v", err)
	}
}
