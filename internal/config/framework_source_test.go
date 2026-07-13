package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_RejectsNonBuiltinFramework(t *testing.T) {
	p := filepath.Join(t.TempDir(), "homonto.toml")
	if err := os.WriteFile(p, []byte("[frameworks.onto]\nsource = \"local:onto\"\nscope = \"user\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(p)
	if err == nil {
		t.Fatal("local: framework source accepted, want a load error")
	}
	if !strings.Contains(err.Error(), "onto") || !strings.Contains(err.Error(), "builtin") {
		t.Errorf("error %q should name the framework and require builtin:", err.Error())
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
effort = "normal"
[models.claude.trivial]
model = "m"
effort = "fast"
[models.opencode.architectural]
model = "m"
effort = "high"
[models.opencode.coding]
model = "m"
effort = "normal"
[models.opencode.trivial]
model = "m"
effort = "fast"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(p); err != nil {
		t.Fatalf("builtin: framework rejected at load: %v", err)
	}
}
