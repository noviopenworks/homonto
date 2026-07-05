package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const sample = `
[mcps.codegraph]
command = ["codegraph", "serve", "--mcp"]

[mcps.brave]
command = ["npx", "-y", "server-brave"]
env = { BRAVE_API_KEY = "${pass:ai/brave}" }
targets = ["claude"]

[skills]
own = ["graphify", "comet"]

[plugins]
claude = ["claude-hud@official"]
opencode = ["@slkiser/opencode-quota"]

[settings.claude]
model = "opus"

[settings.opencode]
model = "anthropic/claude-opus-4-8"
`

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "homonto.toml")
	if err := os.WriteFile(p, []byte(sample), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := Load(p)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got := c.MCPs["codegraph"].Command; len(got) != 3 || got[0] != "codegraph" {
		t.Fatalf("codegraph command = %v", got)
	}
	if got := c.MCPs["brave"].Env["BRAVE_API_KEY"]; got != "${pass:ai/brave}" {
		t.Fatalf("brave env = %q", got)
	}
	if got := c.MCPs["codegraph"].TargetsOrAll(); len(got) != 2 {
		t.Fatalf("default targets = %v", got)
	}
	if got := c.MCPs["brave"].TargetsOrAll(); len(got) != 1 || got[0] != "claude" {
		t.Fatalf("brave targets = %v", got)
	}
	if c.Settings.Claude["model"] != "opus" {
		t.Fatalf("claude model = %v", c.Settings.Claude["model"])
	}
	if len(c.Skills.Own) != 2 {
		t.Fatalf("skills = %v", c.Skills.Own)
	}
}

func TestLoadMissingFile(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "nope.toml")); err == nil {
		t.Fatal("expected error for missing file")
	}
}

// TestLoadRejectsBadSkillNames reproduces the review's traversal finding:
// own = ["../../../escaped"] must be a load-time error, not a symlink
// planted outside $HOME. Every non-bare-directory-name entry is rejected.
func TestLoadRejectsBadSkillNames(t *testing.T) {
	for _, bad := range []string{"../evil", "..", ".", "", "a/b", `a\b`, "/abs"} {
		dir := t.TempDir()
		p := filepath.Join(dir, "homonto.toml")
		doc := "[skills]\nown = [" + strconv.Quote(bad) + "]\n"
		if err := os.WriteFile(p, []byte(doc), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := Load(p)
		if err == nil {
			t.Fatalf("skill name %q accepted; want load error", bad)
		}
		if !strings.Contains(err.Error(), strconv.Quote(bad)) {
			t.Fatalf("error for %q does not name the entry: %v", bad, err)
		}
	}
}
