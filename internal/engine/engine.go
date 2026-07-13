package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/noviopenworks/homonto/internal/adapter"
	"github.com/noviopenworks/homonto/internal/adapter/registry"
	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/secret"
	"github.com/noviopenworks/homonto/internal/state"
)

// Engine wires config, adapters, secret resolver, and state for plan/apply.
type Engine struct {
	Cfg                 *config.Config
	Adapters            []adapter.Adapter
	State               *state.State
	StateDir            string
	ContentDir          string
	CatalogRoot         string // materialized builtin catalog root (<stateDir>/catalog/skills)
	CommandCatalogRoot  string // materialized builtin command root (<stateDir>/catalog/commands)
	SubagentCatalogRoot string // materialized builtin subagent root (<stateDir>/catalog/subagents)
	RemoteRoot          string // materialized remote content root (<stateDir>/remote)
	RemoteCacheRoot     string // content-addressed remote cache (<stateDir>/cache/remote)
	Home                string
	ProjectRoot         string // directory of homonto.toml; skill-scope project root
	Resolver            *secret.Resolver
	// Warnings collects non-fatal per-adapter failures from the last Plan (e.g.
	// an unparseable tool file); other tools still proceed.
	Warnings []string
}

// Build loads config and wires both adapters. home is $HOME; contentDir is the
// local provider root; state lives in <repo>/.homonto next to the config.
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
	// The project root anchors project-scope skill installs — the same directory
	// that already anchors homonto/ and .homonto/ (the config file's directory).
	projectRoot, err := filepath.Abs(filepath.Dir(configPath))
	if err != nil {
		return nil, err
	}
	stateDir := filepath.Join(filepath.Dir(configPath), ".homonto")
	catalogDir := filepath.Join(stateDir, "catalog", "skills")
	commandCatalogDir := filepath.Join(stateDir, "catalog", "commands")
	subagentCatalogDir := filepath.Join(stateDir, "catalog", "subagents")
	remoteRoot := filepath.Join(stateDir, "remote")
	remoteSubagentDir := filepath.Join(remoteRoot, "subagents")
	remoteCacheRoot := filepath.Join(stateDir, "cache", "remote")
	st, err := state.Load(stateDir)
	if err != nil {
		return nil, err
	}
	return &Engine{
		Cfg: cfg,
		Adapters: registry.Builtins().Build(registry.Deps{
			Home:               home,
			ContentDir:         contentDir,
			ProjectRoot:        projectRoot,
			CatalogDir:         catalogDir,
			CommandCatalogDir:  commandCatalogDir,
			SubagentCatalogDir: subagentCatalogDir,
			RemoteSubagentDir:  remoteSubagentDir,
		}),
		State:               st,
		StateDir:            stateDir,
		ContentDir:          contentDir,
		CatalogRoot:         catalogDir,
		CommandCatalogRoot:  commandCatalogDir,
		SubagentCatalogRoot: subagentCatalogDir,
		RemoteRoot:          remoteRoot,
		RemoteCacheRoot:     remoteCacheRoot,
		Home:                home,
		ProjectRoot:         projectRoot,
		Resolver:            secret.NewResolver(),
	}, nil
}

// CatalogDir returns the materialized builtin catalog root.
func (e *Engine) CatalogDir() string { return e.CatalogRoot }

// CommandDir returns the materialized builtin command root.
func (e *Engine) CommandDir() string { return e.CommandCatalogRoot }

// SubagentDir returns the materialized builtin subagent root.
func (e *Engine) SubagentDir() string { return e.SubagentCatalogRoot }

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
// before any write on error), then apply each adapter, saving state after each
// successful adapter so a later failure never loses an earlier one's record.
func (e *Engine) Apply(sets []adapter.ChangeSet) error {
	// Fail closed on a malformed plan before any side effect: an unknown tool
	// (otherwise silently skipped below) or an operation with an undefined action
	// (otherwise a silent no-op) must abort — never quietly drop a change to a
	// user's config files.
	knownTools := make(map[string]bool, len(e.Adapters))
	for _, a := range e.Adapters {
		knownTools[a.Name()] = true
	}
	for _, cs := range sets {
		if err := cs.Validate(knownTools); err != nil {
			return err
		}
	}
	for _, cs := range sets {
		for _, c := range cs.Changes {
			// Deletes carry no New value; nothing to resolve. Adopt is non-secret
			// by construction (Plan only emits it for a value without a ${...} ref),
			// so it too has nothing to resolve — the adapter's Apply records its
			// state hash directly from the already-matching on-disk value.
			if c.Action == "noop" || c.Action == "delete" || c.Action == "adopt" {
				continue
			}
			if _, err := e.Resolver.Resolve(c.New); err != nil {
				return err
			}
		}
	}
	// Resolve, verify, and materialize remote sources before any adapter links
	// them. This fetches → validates → pin-matches → caches, aborting the whole
	// apply before any adapter write if any remote resource fails closed.
	if err := e.materializeRemotes(); err != nil {
		return err
	}
	// Materialize builtin skills before any adapter links them, so no symlink is
	// created ahead of its target.
	if err := e.materializeCatalog(); err != nil {
		return err
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
		// Name the tool in every per-adapter failure: with several adapters an
		// unwrapped error leaves the user guessing which file broke.
		if err := a.Apply(e.Cfg, cs, e.Resolver, e.State); err != nil {
			return fmt.Errorf("%s: %w", cs.Tool, err)
		}
		// Persist immediately: a partial apply must keep the record of every
		// adapter that already wrote its files.
		if err := e.State.Save(e.StateDir); err != nil {
			return fmt.Errorf("%s: save state: %w", cs.Tool, err)
		}
	}
	return e.State.Save(e.StateDir)
}

// materializeCatalog extracts the builtin skills, commands, and subagents the
// config declares into CatalogRoot, CommandCatalogRoot, and
// SubagentCatalogRoot, version-gated: it is a no-op when the recorded catalog
// version matches the embedded one AND every skill dir, command file, and
// subagent file already exists. The version is recorded (and state saved)
// only after skills, commands, AND subagents all materialize, so an
// interrupted extraction re-materializes on the next apply.
func (e *Engine) materializeCatalog() error {
	skillSet := map[string]bool{}
	cmdSet := map[string]bool{}
	subSet := map[string]bool{}
	for _, tool := range []string{"claude", "opencode"} {
		sEntries, err := e.Cfg.ExpandedSkillEntriesForTool(tool)
		if err != nil {
			return err
		}
		for _, entry := range sEntries {
			if strings.HasPrefix(entry.Resource.Source, "builtin:") {
				skillSet[strings.TrimPrefix(entry.Resource.Source, "builtin:")] = true
			}
		}
		cEntries, err := e.Cfg.ExpandedCommandEntriesForTool(tool)
		if err != nil {
			return err
		}
		for _, entry := range cEntries {
			if strings.HasPrefix(entry.Resource.Source, "builtin:") {
				cmdSet[strings.TrimPrefix(entry.Resource.Source, "builtin:")] = true
			}
		}
		saEntries, err := e.Cfg.ExpandedSubagentEntriesForTool(tool)
		if err != nil {
			return err
		}
		for _, entry := range saEntries {
			if strings.HasPrefix(entry.Resource.Source, "builtin:") {
				subSet[strings.TrimPrefix(entry.Resource.Source, "builtin:")] = true
			}
		}
	}
	if len(skillSet) == 0 && len(cmdSet) == 0 && len(subSet) == 0 {
		return nil
	}
	// Build the catalog including the config's local frameworks so a
	// local:<path> framework's resources materialize (from their own FS) into
	// the catalog root exactly like a builtin's. With no local frameworks this
	// is the embedded singleton, identical to catalog.New().
	cl, err := e.Cfg.FrameworkCatalog()
	if err != nil {
		return err
	}
	skillNames := make([]string, 0, len(skillSet))
	for n := range skillSet {
		skillNames = append(skillNames, n)
	}
	sort.Strings(skillNames)
	cmdNames := make([]string, 0, len(cmdSet))
	for n := range cmdSet {
		cmdNames = append(cmdNames, n)
	}
	sort.Strings(cmdNames)
	subNames := make([]string, 0, len(subSet))
	for n := range subSet {
		subNames = append(subNames, n)
	}
	sort.Strings(subNames)

	if e.State.CatalogVersionRecorded() == cl.Version() &&
		allSkillDirsExist(e.CatalogRoot, skillNames) &&
		allCommandFilesExist(e.CommandCatalogRoot, cmdNames) &&
		allSubagentFilesExist(e.SubagentCatalogRoot, subNames) {
		return nil
	}
	if err := cl.Materialize(e.CatalogRoot, skillNames); err != nil {
		return err
	}
	if err := cl.MaterializeCommands(e.CommandCatalogRoot, cmdNames); err != nil {
		return err
	}
	if err := cl.MaterializeSubagents(e.SubagentCatalogRoot, subNames); err != nil {
		return err
	}
	e.State.SetCatalogVersion(cl.Version())
	// Save immediately so a later adapter failure still records the completed
	// materialization.
	return e.State.Save(e.StateDir)
}

func allSkillDirsExist(root string, names []string) bool {
	for _, n := range names {
		fi, err := os.Stat(filepath.Join(root, n))
		if err != nil || !fi.IsDir() {
			return false
		}
	}
	return true
}

func allCommandFilesExist(root string, names []string) bool {
	for _, n := range names {
		fi, err := os.Stat(filepath.Join(root, n+".md"))
		if err != nil || fi.IsDir() {
			return false
		}
	}
	return true
}

func allSubagentFilesExist(root string, names []string) bool {
	for _, n := range names {
		fi, err := os.Stat(filepath.Join(root, n+".md"))
		if err != nil || fi.IsDir() {
			return false
		}
	}
	return true
}
