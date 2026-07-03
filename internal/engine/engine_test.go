package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/noviopenworks/homonto/internal/secret"
)

const cfgTOML = `
[mcps.brave]
command = ["npx","server-brave"]
env = { K = "${MISSING_VAR}" }
targets = ["claude"]
`

func TestApplyAbortsBeforeWritingOnMissingSecret(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	cfgPath := filepath.Join(repo, "homonto.toml")
	os.WriteFile(cfgPath, []byte(cfgTOML), 0o644)

	e, err := Build(cfgPath, home, filepath.Join(repo, "content"))
	if err != nil {
		t.Fatal(err)
	}
	e.Resolver = &secret.Resolver{Getenv: func(string) string { return "" }, Pass: func(string) (string, error) { return "", nil }}

	sets, err := e.Plan()
	if err != nil {
		t.Fatal(err)
	}
	if err := e.Apply(sets); err == nil {
		t.Fatal("expected apply to fail on missing secret")
	}
	if _, err := os.Stat(filepath.Join(home, ".claude.json")); !os.IsNotExist(err) {
		t.Fatal("apply wrote a file despite secret failure (not two-phase)")
	}
	if _, err := os.Stat(filepath.Join(repo, ".homonto", "state.json")); !os.IsNotExist(err) {
		t.Fatal("state written despite secret failure")
	}
}
