package engine

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/noviopenworks/homonto/internal/adapter"
	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/secret"
	"github.com/noviopenworks/homonto/internal/state"
)

type failingAdapter struct{}

func (failingAdapter) Name() string { return "boom" }
func (failingAdapter) Plan(*config.Config, *state.State) (adapter.ChangeSet, error) {
	return adapter.ChangeSet{Tool: "boom"}, nil
}
func (failingAdapter) Apply(adapter.ChangeSet, *secret.Resolver, *state.State) error {
	return errors.New("boom: apply failed")
}

// Deep review: state was saved only after ALL adapters succeeded, so a partial
// apply left no record at all — the next run treated already-written values as
// unknown provenance. State must be saved after each adapter's successful
// Apply, so an earlier adapter's record survives a later adapter's failure.
func TestPartialApplyPersistsEarlierAdapterState(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	cfgPath := filepath.Join(repo, "homonto.toml")
	os.WriteFile(cfgPath, []byte("[mcps.codegraph]\ncommand = [\"codegraph\",\"serve\"]\ntargets = [\"claude\"]\n"), 0o644)

	e, err := Build(cfgPath, home, filepath.Join(repo, "content"))
	if err != nil {
		t.Fatal(err)
	}
	e.Resolver = &secret.Resolver{Getenv: os.Getenv, Pass: func(string) (string, error) { return "", nil }}
	e.Adapters = append(e.Adapters, failingAdapter{})

	sets, err := e.Plan()
	if err != nil {
		t.Fatal(err)
	}
	if err := e.Apply(sets); err == nil {
		t.Fatal("expected apply to fail on the failing adapter")
	}
	st, err := state.Load(filepath.Join(repo, ".homonto"))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := st.Get("claude", "mcp.codegraph"); !ok {
		t.Fatal("claude's applied state lost because a later adapter failed")
	}
}
