package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLoad_RejectsUnsupportedFrameworkSource: a framework source that is neither
// builtin: nor local: (a bare name, or a remote: URL) expands nothing and is
// rejected loudly at load (F35). local: frameworks are accepted post-E1 — see
// TestLoad_AcceptsLocalFramework.
func TestLoad_RejectsUnsupportedFrameworkSource(t *testing.T) {
	for _, src := range []string{"onto", "remote:https://example.com/onto"} {
		p := filepath.Join(t.TempDir(), "homonto.toml")
		if err := os.WriteFile(p, []byte("[frameworks.onto]\nsource = \""+src+"\"\nscope = \"user\"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := Load(p)
		if err == nil {
			t.Fatalf("framework source %q accepted, want a load error", src)
		}
		if !strings.Contains(err.Error(), "onto") {
			t.Errorf("error %q should name the framework", err.Error())
		}
	}
}

// TestLoad_AcceptsLocalFramework: a [frameworks.X] source="local:<path>" is
// accepted at load post-E1. Validation checks only the source shape; the local
// framework root is resolved later at expansion/materialization time.
func TestLoad_AcceptsLocalFramework(t *testing.T) {
	p := filepath.Join(t.TempDir(), "homonto.toml")
	if err := os.WriteFile(p, []byte("[frameworks.myfw]\nsource = \"local:./myfw\"\nscope = \"user\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(p); err != nil {
		t.Fatalf("local: framework rejected at load: %v", err)
	}
}

func TestLoad_AcceptsBuiltinFramework(t *testing.T) {
	p := filepath.Join(t.TempDir(), "homonto.toml")
	if err := os.WriteFile(p, []byte(`[frameworks.onto]
source = "builtin:onto"
scope = "user"
[models.claude.architectural]
model = "m"
effort = "high"
[models.claude.coding]
model = "m"
effort = "medium"
[models.claude.trivial]
model = "m"
effort = "low"
[models.opencode.architectural]
model = "m"
[models.opencode.coding]
model = "m"
[models.opencode.trivial]
model = "m"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(p); err != nil {
		t.Fatalf("builtin: framework rejected at load: %v", err)
	}
}
