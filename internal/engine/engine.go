package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/noviopenworks/homonto/internal/adapter"
	"github.com/noviopenworks/homonto/internal/adapter/registry"
	"github.com/noviopenworks/homonto/internal/agentfm"
	"github.com/noviopenworks/homonto/internal/catalog"
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
	// HomontoVersion is the running binary version, set by the CLI. When set, Plan
	// enforces each declared framework's [compat].homonto range fail-closed; empty
	// (tests/unstamped) skips the check.
	HomontoVersion string
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
	// Anchor state (and the materialized catalog under it) on the absolute
	// projectRoot, not filepath.Dir(configPath): with the default relative
	// --config, the latter is "." and every catalog-skill symlink target would
	// be stored as ".homonto/catalog/skills/<name>" — relative to the *link's*
	// directory (e.g. .opencode/skills/), which dangles. contentDir is
	// absolutized above for the same reason; stateDir must match.
	stateDir := filepath.Join(projectRoot, ".homonto")
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
	e := &Engine{
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
	}
	// Resolve any [frameworks.X] source="remote:<url>" through the trust pipeline
	// now that the cache/lock/revocation paths are known, and inject the verified
	// cache dirs so BOTH Plan (framework expansion) and materializeCatalog (which
	// builds via Cfg.FrameworkCatalog()) overlay the remote framework roots. A
	// config with no remote frameworks resolves nothing (no network); a bad,
	// mismatched, or revoked digest fails closed here and aborts Build.
	dirs, err := e.resolveRemoteFrameworks()
	if err != nil {
		return nil, err
	}
	if len(dirs) > 0 {
		cfg.SetRemoteFrameworkDirs(dirs)
	}
	return e, nil
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
	if err := e.checkFrameworkCompat(); err != nil {
		return nil, err
	}
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
	e.recordVersions()
	return e.State.Save(e.StateDir)
}

// recordVersions writes down, in state, the binary and framework versions behind
// this apply — so `homonto update` can report the transition and `onto` can
// detect a binary/framework skew. Best-effort: a catalog that will not load
// leaves framework versions untouched rather than failing the completed apply.
func (e *Engine) recordVersions() {
	e.State.SetHomontoVersion(e.HomontoVersion)
	cl, err := e.Cfg.FrameworkCatalog()
	if err != nil {
		return
	}
	for name, r := range e.Cfg.Frameworks {
		catName, ok := config.FrameworkCatalogName(name, r.Source)
		if !ok {
			continue
		}
		if v, ok := cl.FrameworkVersion(catName); ok {
			e.State.SetFrameworkVersion(name, v)
		}
	}
}

// subagentRenderContext builds the per-tool agentfm render context: each role's
// tier default from [models.<tool>.<role>], plus any per-subagent override from
// [subagents.<name>.<tool>]. A subagent's neutral `homonto: role` then stamps
// the right tool-native model, variant, and effort. A tool with no routes yields
// an empty map (agents inherit the tool's default model).
//
// Overrides are keyed by the subagent's CATALOG name, not its config key,
// because materialization writes one rendered file per catalog name — two
// declarations of the same builtin share that file. Config validation rejects
// conflicting overrides on one source, so resolving by catalog name here is
// unambiguous by the time we run.
func (e *Engine) subagentRenderContext() map[string]agentfm.RenderContext {
	roleSpecs := func(routes map[string]config.ModelRoute) map[string]agentfm.ModelSpec {
		m := map[string]agentfm.ModelSpec{}
		for role, r := range routes {
			m[role] = agentfm.ModelSpec{Model: r.Model, Variant: r.Variant, Effort: r.Effort}
		}
		return m
	}
	overrides := func(pick func(config.Subagent) config.ModelRoute) map[string]agentfm.ModelSpec {
		m := map[string]agentfm.ModelSpec{}
		for key, sa := range e.Cfg.Subagents {
			r := pick(sa)
			if r.Model == "" && r.Variant == "" && r.Effort == "" {
				continue
			}
			// Resolve to the CATALOG name, which is what materialization renders
			// per file. A declared entry carries it in its builtin: source; a
			// tune-only entry names it directly (it retunes a framework's agent,
			// whose config key IS its catalog name).
			name := key
			if !sa.IsTuneOnly() {
				cat, ok := config.SubagentCatalogName(sa.Source)
				if !ok {
					continue // local:/remote: content is not catalog-keyed
				}
				name = cat
			}
			m[name] = agentfm.ModelSpec{Model: r.Model, Variant: r.Variant, Effort: r.Effort}
		}
		return m
	}
	return map[string]agentfm.RenderContext{
		"claude": {
			Roles:     roleSpecs(e.Cfg.Models.Claude),
			Overrides: overrides(func(s config.Subagent) config.ModelRoute { return s.Claude }),
		},
		"opencode": {
			Roles:     roleSpecs(e.Cfg.Models.OpenCode),
			Overrides: overrides(func(s config.Subagent) config.ModelRoute { return s.OpenCode }),
		},
	}
}

// materializeCatalog extracts the builtin skills, commands, and subagents the
// config declares into CatalogRoot, CommandCatalogRoot, and SubagentCatalogRoot.
// It is a no-op only when planCatalog finds every input unchanged: the recorded
// catalog version matches the embedded one, the subagent render fingerprint
// matches the config's model routes, and every file a materialize would write
// already exists. The version and fingerprint are recorded (and state saved)
// only after skills, commands, AND subagents all materialize, so an interrupted
// extraction re-materializes on the next apply.
func (e *Engine) materializeCatalog() error {
	p, err := e.planCatalog()
	if err != nil {
		return err
	}
	if p == nil || p.upToDate {
		return nil
	}
	if err := p.cl.Materialize(e.CatalogRoot, p.skills); err != nil {
		return err
	}
	if err := p.cl.MaterializeCommands(e.CommandCatalogRoot, p.commands); err != nil {
		return err
	}
	if err := p.cl.MaterializeSubagents(e.SubagentCatalogRoot, p.subagents, p.renderCtx); err != nil {
		return err
	}
	e.State.SetCatalogVersion(p.cl.Version())
	e.State.SetSubagentRenderFingerprint(p.fingerprint)
	// Save immediately so a later adapter failure still records the completed
	// materialization.
	return e.State.Save(e.StateDir)
}

// catalogPlan is what a materialize would extract, and whether it need bother.
type catalogPlan struct {
	cl          *catalog.Catalog
	skills      []string
	commands    []string
	subagents   []string
	renderCtx   map[string]agentfm.RenderContext
	fingerprint string
	upToDate    bool
}

// CatalogNeedsMaterialize reports whether a materialize would do real work. The
// CLI needs this because a catalog file's symlink target is name-based, so
// stale, missing, or mis-rendered catalog content leaves the projection plan
// empty — and an empty plan otherwise skips apply entirely, stranding the
// content forever. (Same shape as the HasRemoteResources carve-out.) An error
// resolving the plan counts as "needs work" so apply runs and surfaces it,
// rather than being silently swallowed here.
func (e *Engine) CatalogNeedsMaterialize() bool {
	p, err := e.planCatalog()
	if err != nil {
		return true
	}
	return p != nil && !p.upToDate
}

// planCatalog resolves the builtin content the config declares and evaluates the
// materialize gate. It returns nil when nothing builtin is declared.
func (e *Engine) planCatalog() (*catalogPlan, error) {
	skillSet := map[string]bool{}
	cmdSet := map[string]bool{}
	subSet := map[string]bool{}
	for _, tool := range []string{"claude", "opencode"} {
		sEntries, err := e.Cfg.ExpandedSkillEntriesForTool(tool)
		if err != nil {
			return nil, err
		}
		for _, entry := range sEntries {
			if strings.HasPrefix(entry.Resource.Source, "builtin:") {
				skillSet[strings.TrimPrefix(entry.Resource.Source, "builtin:")] = true
			}
		}
		cEntries, err := e.Cfg.ExpandedCommandEntriesForTool(tool)
		if err != nil {
			return nil, err
		}
		for _, entry := range cEntries {
			if strings.HasPrefix(entry.Resource.Source, "builtin:") {
				cmdSet[strings.TrimPrefix(entry.Resource.Source, "builtin:")] = true
			}
		}
		saEntries, err := e.Cfg.ExpandedSubagentEntriesForTool(tool)
		if err != nil {
			return nil, err
		}
		for _, entry := range saEntries {
			if strings.HasPrefix(entry.Resource.Source, "builtin:") {
				subSet[strings.TrimPrefix(entry.Resource.Source, "builtin:")] = true
			}
		}
	}
	if len(skillSet) == 0 && len(cmdSet) == 0 && len(subSet) == 0 {
		return nil, nil
	}
	// Build the catalog including the config's local frameworks so a
	// local:<path> framework's resources materialize (from their own FS) into
	// the catalog root exactly like a builtin's. With no local frameworks this
	// is the embedded singleton, identical to catalog.New().
	cl, err := e.Cfg.FrameworkCatalog()
	if err != nil {
		return nil, err
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

	// Gate on the catalog version AND the render fingerprint: a subagent's
	// rendered frontmatter is derived from config (the model routes), so the
	// version alone would freeze rendered agents at their old model whenever a
	// route changes — the catalog is identical, only the config moved.
	renderCtx := e.subagentRenderContext()
	fingerprint := subagentRenderFingerprint(renderCtx)
	upToDate := e.State.CatalogVersionRecorded() == cl.Version() &&
		e.State.SubagentRenderFingerprintRecorded() == fingerprint &&
		allSkillDirsExist(e.CatalogRoot, skillNames) &&
		allCommandFilesExist(e.CommandCatalogRoot, cmdNames) &&
		allSubagentFilesExist(e.SubagentCatalogRoot, subNames, cl, renderCtx)
	return &catalogPlan{
		cl:          cl,
		skills:      skillNames,
		commands:    cmdNames,
		subagents:   subNames,
		renderCtx:   renderCtx,
		fingerprint: fingerprint,
		upToDate:    upToDate,
	}, nil
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

// subagentRenderFingerprint digests every render input — each tool's role tiers
// AND per-subagent overrides, model + variant + effort — deterministically, so
// the materialize gate re-renders exactly when something the agents are stamped
// from actually changed. Every field the render reads must be digested here: one
// omitted field is one config edit that silently never reaches the agent.
//
// Sorted keys keep it stable across map iteration order; delimited fields keep
// it unambiguous across values (an "a"+"bc" / "ab"+"c" collision would skip the
// very re-render this gate exists to trigger).
func subagentRenderFingerprint(ctx map[string]agentfm.RenderContext) string {
	h := sha256.New()
	digestSpecs := func(kind, tool string, specs map[string]agentfm.ModelSpec) {
		keys := make([]string, 0, len(specs))
		for k := range specs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			s := specs[k]
			fmt.Fprintf(h, "%s\x00%s\x00%s\x00%s\x00%s\x00%s\x00", kind, tool, k, s.Model, s.Variant, s.Effort)
		}
	}
	tools := make([]string, 0, len(ctx))
	for tool := range ctx {
		tools = append(tools, tool)
	}
	sort.Strings(tools)
	for _, tool := range tools {
		digestSpecs("role", tool, ctx[tool].Roles)
		digestSpecs("override", tool, ctx[tool].Overrides)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// allSubagentFilesExist reports whether every file a materialize would write is
// present — the shared anchor AND each per-tool rendered variant. Checking only
// the <name>.md anchor would leave a deleted variant unrepaired: the anchor
// still exists, so the gate short-circuits and the tool keeps a symlink pointing
// at a file that is never rewritten.
func allSubagentFilesExist(root string, names []string, cl *catalog.Catalog, renderCtx map[string]agentfm.RenderContext) bool {
	for _, n := range names {
		files, err := cl.SubagentFiles(n, renderCtx)
		if err != nil {
			return false
		}
		for _, f := range files {
			fi, err := os.Stat(filepath.Join(root, f))
			if err != nil || fi.IsDir() {
				return false
			}
		}
	}
	return true
}
