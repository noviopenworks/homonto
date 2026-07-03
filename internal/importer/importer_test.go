package importer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImportRedactsSecretsInEnv(t *testing.T) {
	home := t.TempDir()
	os.WriteFile(filepath.Join(home, ".claude.json"), []byte(`{
	  "mcpServers": {
	    "brave": {"command":["npx","server-brave"],"env":{"BRAVE_API_KEY":"sk-secret-123"}}
	  }
	}`), 0o644)

	c, warnings, err := Import(home)
	if err != nil {
		t.Fatal(err)
	}
	got := c.MCPs["brave"].Env["BRAVE_API_KEY"]
	if !strings.HasPrefix(got, "${pass:") {
		t.Fatalf("secret not redacted: %q", got)
	}
	if strings.Contains(got, "sk-secret-123") {
		t.Fatal("literal secret leaked into config")
	}
	if len(warnings) == 0 {
		t.Fatal("expected a warning about the redacted secret")
	}

	out, err := MarshalTOML(c)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "sk-secret-123") {
		t.Fatal("literal secret leaked into TOML output")
	}
}

func TestImportKeepsNonSecretValues(t *testing.T) {
	home := t.TempDir()
	os.WriteFile(filepath.Join(home, ".claude.json"), []byte(`{
	  "mcpServers": {"cg": {"command":["codegraph","serve"],"env":{"MODE":"fast"}}}
	}`), 0o644)
	c, _, err := Import(home)
	if err != nil {
		t.Fatal(err)
	}
	if c.MCPs["cg"].Env["MODE"] != "fast" {
		t.Fatalf("non-secret value altered: %q", c.MCPs["cg"].Env["MODE"])
	}
}
