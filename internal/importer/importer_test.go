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

func TestImportReadsRealSchemaCommandAndArgs(t *testing.T) {
	// Real Claude Code files store command as a string with a separate args
	// array; import must not drop the args.
	home := t.TempDir()
	os.WriteFile(filepath.Join(home, ".claude.json"), []byte(`{
	  "mcpServers": {
	    "brave": {"type":"stdio","command":"npx","args":["-y","@modelcontextprotocol/server-brave-search"]}
	  }
	}`), 0o644)

	c, _, err := Import(home)
	if err != nil {
		t.Fatal(err)
	}
	got := c.MCPs["brave"].Command
	want := []string{"npx", "-y", "@modelcontextprotocol/server-brave-search"}
	if len(got) != len(want) {
		t.Fatalf("args dropped: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("command[%d] = %q, want %q (full: %v)", i, got[i], want[i], got)
		}
	}
	out, err := MarshalTOML(c)
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range want {
		if !strings.Contains(string(out), w) {
			t.Fatalf("TOML missing command element %q:\n%s", w, out)
		}
	}
}

func TestRedactionCoverage(t *testing.T) {
	hits := []struct{ key, val string }{
		{"MY_PASSWORD", "hunter2"},
		{"APP_SECRET", "whatever"},
		{"GCP_CREDENTIALS", "whatever"},
		{"DATABASE_URL", "postgres://u:p@h/db"},
		{"X", "glpat-abc123"},
		{"X", "npm_abc123"},
		{"X", "AIzaSyExample"},
		{"X", "Bearer abc.def"},
		{"X", "xoxb-123"},
	}
	for _, tc := range hits {
		if _, hit := redact("srv", tc.key, tc.val); !hit {
			t.Errorf("redact(%q, %q) should hit", tc.key, tc.val)
		}
	}
	misses := []struct{ key, val string }{
		{"DEBUG", "true"},
		{"MODE", "fast"},
		{"PASSWORD_HINT_ENABLED", "yes"}, // suffix match only, not substring
	}
	for _, tc := range misses {
		if got, hit := redact("srv", tc.key, tc.val); hit {
			t.Errorf("redact(%q, %q) should not hit, got %q", tc.key, tc.val, got)
		}
	}
}

func TestImportRedactsPasswordValueEndToEnd(t *testing.T) {
	home := t.TempDir()
	os.WriteFile(filepath.Join(home, ".claude.json"), []byte(`{
	  "mcpServers": {
	    "db": {"type":"stdio","command":"db-mcp","env":{"MY_PASSWORD":"hunter2","DEBUG":"true"}}
	  }
	}`), 0o644)
	c, warnings, err := Import(home)
	if err != nil {
		t.Fatal(err)
	}
	if got := c.MCPs["db"].Env["MY_PASSWORD"]; !strings.HasPrefix(got, "${pass:") {
		t.Fatalf("MY_PASSWORD not redacted: %q", got)
	}
	if got := c.MCPs["db"].Env["DEBUG"]; got != "true" {
		t.Fatalf("DEBUG must survive untouched, got %q", got)
	}
	if len(warnings) != 1 {
		t.Fatalf("expected exactly one redaction warning, got %v", warnings)
	}
	out, _ := MarshalTOML(c)
	if strings.Contains(string(out), "hunter2") {
		t.Fatal("literal password leaked into TOML output")
	}
}

func TestImportWarnsOnUnreadableClaudeJSON(t *testing.T) {
	home := t.TempDir()
	// A directory at the file's path makes ReadFile fail with a non-not-exist
	// error; import must surface that instead of silently skipping.
	os.MkdirAll(filepath.Join(home, ".claude.json"), 0o755)
	_, warnings, err := Import(home)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) == 0 {
		t.Fatal("unreadable .claude.json must produce a warning, got none")
	}
}

func TestImportSilentWhenClaudeJSONAbsent(t *testing.T) {
	_, warnings, err := Import(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("missing file is not an error condition, got %v", warnings)
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
