package engine

import (
	"fmt"
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
	Home       string
	Resolver   *secret.Resolver
	// Warnings collects non-fatal per-adapter failures from the last Plan (e.g.
	// an unparseable tool file); other tools still proceed.
	Warnings []string
}

// Build loads config and wires both adapters. home is $HOME; contentDir holds
// owned content; state lives in <repo>/.homonto next to the config.
func Build(configPath, home, contentDir string) (*Engine, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, err
	}
	// A relative content dir is relative to the config file, not the shell
	// working directory — symlink targets must stay valid from anywhere.
	if !filepath.IsAbs(contentDir) {
		base, err := filepath.Abs(filepath.Dir(configPath))
		if err != nil {
			return nil, err
		}
		contentDir = filepath.Join(base, contentDir)
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
		Home:       home,
		Resolver:   secret.NewResolver(),
	}, nil
}

// Plan runs each adapter's Plan. An adapter that fails (e.g. its tool file is
// unparseable) is skipped with a warning so the other tools still proceed; its
// file is never written. Warnings from the run are recorded on e.Warnings.
func (e *Engine) Plan() ([]adapter.ChangeSet, error) {
	e.Warnings = nil
	var sets []adapter.ChangeSet
	for _, a := range e.Adapters {
		cs, err := a.Plan(e.Cfg, e.State)
		if err != nil {
			e.Warnings = append(e.Warnings, fmt.Sprintf("%s skipped: %v", a.Name(), err))
			continue
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
	// Match each planned set to its adapter by tool name (Plan may have skipped
	// some adapters, so indexes need not line up).
	byName := map[string]adapter.Adapter{}
	for _, a := range e.Adapters {
		byName[a.Name()] = a
	}
	for _, cs := range sets {
		a, ok := byName[cs.Tool]
		if !ok {
			continue
		}
		if err := a.Apply(cs, e.Resolver, e.State); err != nil {
			return err
		}
	}
	return e.State.Save(e.StateDir)
}
