package engine

import (
	"path/filepath"

	"github.com/noviopenworks/homonto/internal/adapter"
	"github.com/noviopenworks/homonto/internal/adapter/claude"
	"github.com/noviopenworks/homonto/internal/adapter/opencode"
	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/secret"
	"github.com/noviopenworks/homonto/internal/state"
)

// Engine wires config, adapters, secret resolver, and state for plan/apply.
type Engine struct {
	Cfg        *config.Config
	Adapters   []adapter.Adapter
	State      *state.State
	StateDir   string
	ContentDir string
	Resolver   *secret.Resolver
}

// Build loads config and wires both adapters. home is $HOME; contentDir holds
// owned content; state lives in <repo>/.homonto next to the config.
func Build(configPath, home, contentDir string) (*Engine, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, err
	}
	stateDir := filepath.Join(filepath.Dir(configPath), ".homonto")
	st, err := state.Load(stateDir)
	if err != nil {
		return nil, err
	}
	return &Engine{
		Cfg:        cfg,
		Adapters:   []adapter.Adapter{claude.New(home, contentDir), opencode.New(home, contentDir)},
		State:      st,
		StateDir:   stateDir,
		ContentDir: contentDir,
		Resolver:   secret.NewResolver(),
	}, nil
}

// Plan runs each adapter's Plan.
func (e *Engine) Plan() ([]adapter.ChangeSet, error) {
	var sets []adapter.ChangeSet
	for _, a := range e.Adapters {
		cs, err := a.Plan(e.Cfg, e.State)
		if err != nil {
			return nil, err
		}
		sets = append(sets, cs)
	}
	return sets, nil
}

// Apply is two-phase: resolve every non-noop change's secrets first (abort
// before any write on error), then apply each adapter, then save state last.
func (e *Engine) Apply(sets []adapter.ChangeSet) error {
	for _, cs := range sets {
		for _, c := range cs.Changes {
			if c.Action == "noop" {
				continue
			}
			if _, err := e.Resolver.Resolve(c.New); err != nil {
				return err
			}
		}
	}
	for i, a := range e.Adapters {
		if err := a.Apply(sets[i], e.Resolver, e.State); err != nil {
			return err
		}
	}
	return e.State.Save(e.StateDir)
}
