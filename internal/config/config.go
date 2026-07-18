package config

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/noviopenworks/homonto/internal/agentfm"
	cat "github.com/noviopenworks/homonto/internal/catalog"
	"github.com/noviopenworks/homonto/internal/remote"
	toml "github.com/pelletier/go-toml/v2"
)

// MCP is a declared MCP server. Env values may hold unresolved ${...} tokens.
type MCP struct {
	Command []string          `toml:"command"`
	Env     map[string]string `toml:"env"`
	Targets []string          `toml:"targets"`
	// Scope selects where the server projects: "user" (default) → the global
	// tool config; "project" → the project-level config each tool merges over
	// it (OpenCode <repo>/opencode.jsonc; Claude <repo>/.mcp.json), so a
	// repository's servers don't run in every other session. Codex is
	// user-scope only (the pilot has no project config).
	Scope string `toml:"scope"`
}

// ScopeOrDefault returns the scope, defaulting to user (the historical
// always-global projection) when unset.
func (m MCP) ScopeOrDefault() string {
	if m.Scope == "" {
		return "user"
	}
	return m.Scope
}

// TargetsOrAll returns the explicit targets, or all tools when none are set.
func (m MCP) TargetsOrAll() []string {
	if len(m.Targets) == 0 {
		return []string{"claude", "opencode"}
	}
	return m.Targets
}

type Resource struct {
	Source  string   `toml:"source"`
	Scope   string   `toml:"scope"`
	Targets []string `toml:"targets"`
	// Digest is the sha256 content pin required when Source is a remote: source.
	Digest string `toml:"digest"`
}

func (r Resource) TargetsOrAll() []string {
	if len(r.Targets) == 0 {
		return []string{"claude", "opencode"}
	}
	return r.Targets
}

type NamedResource struct {
	Name     string
	Resource Resource
	// Mode is the subagent projection mode ("link"|"copy"); empty means link and
	// is the only value skills/commands ever carry. It lets the adapters route a
	// copy-mode subagent to the content-file path instead of the symlink path.
	Mode string
}

// Subagent is a declarative agent projected by `apply` (distinct from the shared
// Resource so it can carry lifecycle fields the reconciliation adds without
// affecting skills/commands). `mode` selects link (symlink, today's behavior) or
// copy (an editable, versioned, mergeable install — landing incrementally);
// `version` is informational until pinning is wired.
type Subagent struct {
	Source  string   `toml:"source"`
	Scope   string   `toml:"scope"`
	Targets []string `toml:"targets"`
	Mode    string   `toml:"mode"`
	Version string   `toml:"version"`
	// Digest is the sha256 content pin ("sha256:<hex>") required when Source is a
	// remote: source and unused otherwise.
	Digest string `toml:"digest"`
	// Claude and OpenCode are per-tool model overrides for THIS subagent,
	// declared as [subagents.<name>.<tool>]. Each field set here wins over the
	// agent's role tier ([models.<tool>.<role>]) field by field, so retuning one
	// agent's effort does not mean restating its model.
	Claude   ModelRoute `toml:"claude"`
	OpenCode ModelRoute `toml:"opencode"`
}

// ModelOverrideFor returns this subagent's override for tool.
func (s Subagent) ModelOverrideFor(tool string) ModelRoute {
	switch tool {
	case "claude":
		return s.Claude
	case "opencode":
		return s.OpenCode
	}
	return ModelRoute{}
}

// IsTuneOnly reports whether this entry only tunes an agent's models rather than
// declaring one: no source, but at least one per-tool model block.
//
// It exists because a framework's subagents may not be re-declared explicitly
// (that collision is an error), which would otherwise leave no way at all to
// retune the model of an agent you installed via [frameworks.*] — the main
// reason to reach for an override. So
//
//	[subagents.onto-skeptic.claude]
//	effort = "max"
//
// with no source is read as "tune onto-skeptic", not "declare it": it projects
// nothing, and never collides with the framework that owns the agent.
func (s Subagent) IsTuneOnly() bool {
	if strings.TrimSpace(s.Source) != "" {
		return false
	}
	return s.Claude != ModelRoute{} || s.OpenCode != ModelRoute{}
}

// SubagentCatalogName returns the builtin catalog name a source resolves to.
// Only builtin: sources have one — local:/remote: content is not catalog-keyed.
func SubagentCatalogName(source string) (string, bool) {
	return strings.CutPrefix(source, "builtin:")
}

func (s Subagent) TargetsOrAll() []string {
	if len(s.Targets) == 0 {
		return []string{"claude", "opencode"}
	}
	return s.Targets
}

// ScopeOrDefault returns the scope, defaulting to project when unset.
func (s Subagent) ScopeOrDefault() string {
	if s.Scope == "" {
		return "project"
	}
	return s.Scope
}

// ModeOrDefault returns the projection mode, defaulting to link (symlink).
func (s Subagent) ModeOrDefault() string {
	if s.Mode == "" {
		return "link"
	}
	return s.Mode
}

// asResource projects the subagent onto the shared Resource shape used by the
// link-based projection pipeline (the only path implemented today).
func (s Subagent) asResource() Resource {
	return Resource{Source: s.Source, Scope: s.ScopeOrDefault(), Targets: s.Targets, Digest: s.Digest}
}

// Agent is a v2 lifecycle-managed agent (distinct from the v1 [subagents]
// symlink Resource): it carries version + mode for update/migration later.
type Agent struct {
	Source  string   `toml:"source"`  // builtin:<name> | local:<name>
	Version string   `toml:"version"` // optional; empty = unpinned
	Targets []string `toml:"targets"` // optional; empty = both tools
	Mode    string   `toml:"mode"`    // optional; copy | link (empty = link)
}

// TargetsOrAll returns the explicit targets, or all tools when none are set.
func (a Agent) TargetsOrAll() []string {
	if len(a.Targets) == 0 {
		return []string{"claude", "opencode"}
	}
	return a.Targets
}

// ModeOrDefault returns the lifecycle mode, defaulting to link when unset.
func (a Agent) ModeOrDefault() string {
	if a.Mode == "" {
		return "link"
	}
	return a.Mode
}

type ModelRoute struct {
	Model   string `toml:"model"`
	Effort  string `toml:"effort"`
	Variant string `toml:"variant"`
}

type ModelConfig struct {
	Claude   map[string]ModelRoute `toml:"claude"`
	OpenCode map[string]ModelRoute `toml:"opencode"`
}

// Plugin is one declared plugin. Source is the tool-native identifier: for
// claude the "name@marketplace" key used in enabledPlugins; for opencode the
// npm package / local plugin path placed in the `plugin` array.
type Plugin struct {
	Source  string         `toml:"source"`
	Enabled *bool          `toml:"enabled"` // nil == true (default enabled)
	Config  map[string]any `toml:"config"`  // non-sensitive per-plugin options
}

// IsEnabled reports whether the plugin is enabled (default true when omitted).
func (p Plugin) IsEnabled() bool { return p.Enabled == nil || *p.Enabled }

type Plugins struct {
	Claude   map[string]Plugin `toml:"claude"`
	OpenCode map[string]Plugin `toml:"opencode"`
}

type Settings struct {
	Claude   map[string]any `toml:"claude"`
	OpenCode map[string]any `toml:"opencode"`
}

// TUI declares per-tool TUI settings projected to a tool-native TUI file. Only
// OpenCode has a separate TUI file (~/.config/opencode/tui.json); Claude's TUI
// settings are ordinary settings.json keys covered by [settings.claude].
type TUI struct {
	OpenCode map[string]any `toml:"opencode"`
}

// Marketplace is one declared Claude plugin marketplace. Source selects which
// locator fields are meaningful: github→Repo, url→URL, git-subdir→URL+Path,
// directory→Path. AutoUpdate is optional (nil == omitted).
type Marketplace struct {
	Source     string `toml:"source"`      // github | url | git-subdir | directory
	Repo       string `toml:"repo"`        // github
	URL        string `toml:"url"`         // url, git-subdir
	Path       string `toml:"path"`        // git-subdir, directory
	AutoUpdate *bool  `toml:"auto_update"` // optional
}

type Marketplaces struct {
	Claude map[string]Marketplace `toml:"claude"`
}

// CurrentConfigSchemaVersion is the homonto.toml schema version this binary
// supports. A config declaring a higher version is rejected fail-closed at load.
const CurrentConfigSchemaVersion = 1

// Config is the tool-agnostic desired state parsed from homonto.toml.
type Config struct {
	// SchemaVersion is the homonto.toml format version. Absent/0 means a legacy
	// (pre-versioning) config and is treated as the current version; a value
	// greater than CurrentConfigSchemaVersion is rejected fail-closed at load so
	// an older binary never silently mis-applies a newer config.
	SchemaVersion int                 `toml:"schema_version,omitempty"`
	MCPs          map[string]MCP      `toml:"mcps"`
	Frameworks    map[string]Resource `toml:"frameworks"`
	Skills        map[string]Resource `toml:"skills"`
	Commands      map[string]Resource `toml:"commands"`
	Subagents     map[string]Subagent `toml:"subagents"`
	Models        ModelConfig         `toml:"models"`
	Plugins       Plugins             `toml:"plugins"`
	Settings      Settings            `toml:"settings"`
	TUI           TUI                 `toml:"tui"`
	Marketplaces  Marketplaces        `toml:"marketplaces"`
	Agents        map[string]Agent    `toml:"agents"`

	// baseDir is the absolute directory of the homonto.toml this config was
	// loaded from. It resolves a [frameworks.X] source="local:<path>" framework
	// root relative to the config file, so local paths need not be threaded
	// through the Expanded* method signatures. Empty for a config not built via
	// Load (e.g. decode in tests): local frameworks then resolve relative to cwd.
	baseDir string

	// remoteFrameworkDirs maps a [frameworks.X] source="remote:<url>" name to the
	// verified cache dir the engine resolved through the remote trust pipeline
	// (fetch → verify → digest-pin → revocation). The engine injects it via
	// SetRemoteFrameworkDirs after resolution so FrameworkCatalog overlays a
	// remote framework root exactly like a local:<path> one. Nil for a config with
	// no remote frameworks (or before resolution): the builtin path is unchanged.
	remoteFrameworkDirs map[string]string
}

// SetRemoteFrameworkDirs injects the verified cache dirs the engine resolved for
// this config's remote frameworks (name → cache dir). FrameworkCatalog then
// overlays each as a framework root keyed by its config name, so both expansion
// (Plan) and catalog materialization (apply) see the remote framework's
// resources projected as builtin:<name>, identical to a local framework.
func (c *Config) SetRemoteFrameworkDirs(dirs map[string]string) {
	c.remoteFrameworkDirs = dirs
}

func (c *Config) SkillEntriesForTool(tool string) []NamedResource {
	return entriesForTool(c.Skills, tool)
}

func (c *Config) CommandEntriesForTool(tool string) []NamedResource {
	return entriesForTool(c.Commands, tool)
}

func (c *Config) SubagentEntriesForTool(tool string) []NamedResource {
	var out []NamedResource
	for name, s := range c.Subagents {
		// A tune-only entry declares no agent — it retunes one a framework owns,
		// so it must not project, and must not count as an explicit declaration
		// that collides with that framework.
		if s.IsTuneOnly() {
			continue
		}
		if containsString(s.TargetsOrAll(), tool) {
			out = append(out, NamedResource{Name: name, Resource: s.asResource(), Mode: s.ModeOrDefault()})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

var (
	catalogOnce sync.Once
	catalogInst *cat.Catalog
	catalogErr  error
)

// loadedCatalog lazily builds the singleton embedded catalog (cheap to index).
func loadedCatalog() (*cat.Catalog, error) {
	catalogOnce.Do(func() { catalogInst, catalogErr = cat.New() })
	return catalogInst, catalogErr
}

// frameworkCatalog returns the catalog used to expand this config's frameworks:
// the embedded builtin catalog overlaid with each [frameworks.X]
// source="local:<path>" as a local single-framework root keyed by X (its path
// resolved relative to the config's baseDir). When the config declares no local
// frameworks it returns the cached embedded singleton unchanged, so a
// builtin-only config expands EXACTLY as before (no per-call re-indexing, no
// behavior change). A local framework's resources index and materialize as
// builtin:<name>, reusing the whole projection path.
func (c *Config) FrameworkCatalog() (*cat.Catalog, error) {
	locals := map[string]fs.FS{}
	for name, fw := range c.Frameworks {
		p, ok := strings.CutPrefix(fw.Source, "local:")
		if !ok {
			continue
		}
		root := p
		if !filepath.IsAbs(root) {
			root = filepath.Join(c.baseDir, p)
		}
		locals[name] = os.DirFS(root)
	}
	// Overlay each remote framework's verified cache dir keyed by its config name,
	// exactly like a local framework root. The engine resolved and digest-verified
	// the dir through the trust pipeline before injecting it, so this path adds no
	// fetch/verify logic — it merges an already-trusted framework root.
	for name, dir := range c.remoteFrameworkDirs {
		locals[name] = os.DirFS(dir)
	}
	if len(locals) == 0 {
		return loadedCatalog()
	}
	return cat.NewWithLocal(locals)
}

// frameworkCatalogName maps a [frameworks.X] declaration to the catalog
// framework name to expand, and reports whether it is expandable. A builtin:<n>
// source expands framework n from the embedded catalog; a local:<path> or
// remote:<url> source expands the framework keyed by the config name X
// (frameworkCatalog indexed the local/remote root under X). Any other source is
// not expandable (false); validation already rejected it at load, so this is
// defensive.
func FrameworkCatalogName(fwName, source string) (string, bool) {
	if n, ok := strings.CutPrefix(source, "builtin:"); ok && n != "" {
		return n, true
	}
	if strings.HasPrefix(source, "local:") {
		return fwName, true
	}
	// A remote framework expands by its config-key name (FrameworkCatalog overlaid
	// its verified cache dir under X), exactly like a local one; its resources are
	// tagged builtin:<name> through the same projection.
	if remote.IsRemoteSource(source) {
		return fwName, true
	}
	return "", false
}

func sameResource(a, b Resource) bool {
	return a.Source == b.Source && a.Scope == b.Scope && slices.Equal(a.Targets, b.Targets)
}

// expandEntriesForTool is the generic per-kind framework-expansion pipeline
// (F43): explicit [<kind>s.X] entries plus, for each framework declaration
// targeting the tool, its transitively expanded resources of the kind — tagged
// builtin:<name> with the framework's scope/targets, merged with the same
// explicit-clash and conflicting-scope/targets rules for every kind. kind fills
// the error text ("skill"/"command"/"subagent"); expand adapts the per-kind
// catalog Expand method to the resource names.
func (c *Config) expandEntriesForTool(tool, kind string, base []NamedResource, expand func(*cat.Catalog, string) ([]string, error)) ([]NamedResource, error) {
	byName := map[string]NamedResource{}
	explicitNames := map[string]bool{}
	for _, e := range base {
		byName[e.Name] = e
		explicitNames[e.Name] = true
	}
	// Deterministic framework iteration order for stable error messages.
	fwNames := make([]string, 0, len(c.Frameworks))
	for name := range c.Frameworks {
		fwNames = append(fwNames, name)
	}
	sort.Strings(fwNames)
	var cl *cat.Catalog
	for _, fwName := range fwNames {
		fwRes := c.Frameworks[fwName]
		catName, ok := FrameworkCatalogName(fwName, fwRes.Source)
		if !ok {
			continue
		}
		if !containsString(fwRes.TargetsOrAll(), tool) {
			continue
		}
		if cl == nil {
			var err error
			if cl, err = c.FrameworkCatalog(); err != nil {
				return nil, err
			}
		}
		names, err := expand(cl, catName)
		if err != nil {
			return nil, fmt.Errorf("config: framework %q: %w", fwName, err)
		}
		for _, name := range names {
			if explicitNames[name] {
				return nil, fmt.Errorf("config: %s %q is declared both explicitly in [%ss] and by framework %q", kind, name, kind, fwName)
			}
			nr := NamedResource{
				Name: name,
				Resource: Resource{
					Source:  "builtin:" + name,
					Scope:   fwRes.Scope,
					Targets: fwRes.Targets,
				},
				Mode: "link", // framework-expanded resources project as symlinks
			}
			if prev, ok := byName[name]; ok {
				if !sameResource(prev.Resource, nr.Resource) {
					return nil, fmt.Errorf("config: %s %q expanded by multiple frameworks with conflicting scope/targets (framework %q)", kind, name, fwName)
				}
				continue
			}
			byName[name] = nr
		}
	}
	out := make([]NamedResource, 0, len(byName))
	for _, nr := range byName {
		out = append(out, nr)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// skillNames/commandNames/subagentNames extract the expanded resource names (the
// only field the pipeline uses) from each kind's catalog Expand result.
func skillNames(e []cat.ExpandedSkill) []string {
	out := make([]string, len(e))
	for i, x := range e {
		out[i] = x.Name
	}
	return out
}
func commandNames(e []cat.ExpandedCommand) []string {
	out := make([]string, len(e))
	for i, x := range e {
		out[i] = x.Name
	}
	return out
}
func subagentNames(e []cat.ExpandedSubagent) []string {
	out := make([]string, len(e))
	for i, x := range e {
		out[i] = x.Name
	}
	return out
}

// ExpandedSkillEntriesForTool returns the effective skills for a tool: explicit
// [skills.X] entries plus, for each [frameworks.<fw>] source="builtin:<fw>"
// targeting the tool, its transitively expanded skills. Each expanded skill
// inherits the framework declaration's scope and targets. A framework skill
// whose name collides with an explicit [skills.X] entry, or with another
// framework's skill under a conflicting declaration, is an error, as is a
// dependency cycle (surfaced from catalog.Expand).
func (c *Config) ExpandedSkillEntriesForTool(tool string) ([]NamedResource, error) {
	return c.expandEntriesForTool(tool, "skill", c.SkillEntriesForTool(tool),
		func(cl *cat.Catalog, n string) ([]string, error) {
			e, err := cl.Expand([]string{n})
			return skillNames(e), err
		})
}

// ExpandedCommandEntriesForTool returns the effective commands for a tool:
// explicit [commands.X] entries plus, for each [frameworks.<fw>]
// source="builtin:<fw>" targeting the tool, its transitively expanded commands.
// Each expanded command inherits the framework declaration's scope and targets.
// A framework command whose name collides with an explicit [commands.X] entry,
// or with another framework's command under a conflicting declaration, is an
// error, as is a dependency cycle (surfaced from catalog.ExpandCommands).
// Collision is command-vs-command only: a command may share a name with a skill.
func (c *Config) ExpandedCommandEntriesForTool(tool string) ([]NamedResource, error) {
	return c.expandEntriesForTool(tool, "command", c.CommandEntriesForTool(tool),
		func(cl *cat.Catalog, n string) ([]string, error) {
			e, err := cl.ExpandCommands([]string{n})
			return commandNames(e), err
		})
}

// ExpandedSubagentEntriesForTool returns the effective subagents for a tool:
// explicit [subagents.X] entries plus, for each [frameworks.<fw>]
// source="builtin:<fw>" targeting the tool, its transitively expanded
// subagents. Each expanded subagent inherits the framework declaration's scope
// and targets. A framework subagent whose name collides with an explicit
// [subagents.X] entry, or with another framework's subagent under a conflicting
// declaration, is an error, as is a dependency cycle (surfaced from
// catalog.ExpandSubagents). Collision is subagent-vs-subagent only.
func (c *Config) ExpandedSubagentEntriesForTool(tool string) ([]NamedResource, error) {
	return c.expandEntriesForTool(tool, "subagent", c.SubagentEntriesForTool(tool),
		func(cl *cat.Catalog, n string) ([]string, error) {
			e, err := cl.ExpandSubagents([]string{n})
			return subagentNames(e), err
		})
}

func (c *Config) EnabledModelTools() []string {
	seen := map[string]bool{}
	// Builtin frameworks enable model routing for their targeted tools (they
	// expand model-routed commands/agents). A local:<path> framework may
	// contribute only skills — like a skills-only config, which needs no models
	// — so it does not by itself force model routes; its expanded resources are
	// validated on their own where model routing actually applies. This keeps the
	// builtin path identical while letting a skill-only local framework load
	// without a [models] block.
	for _, r := range c.Frameworks {
		if !strings.HasPrefix(r.Source, "builtin:") {
			continue
		}
		for _, target := range r.TargetsOrAll() {
			seen[target] = true
		}
	}
	for _, r := range c.Commands {
		for _, target := range r.TargetsOrAll() {
			seen[target] = true
		}
	}
	for _, s := range c.Subagents {
		// A tune-only entry projects no agent, so it enables no tool: it only
		// retunes an agent something else already installed. Counting it would
		// demand model routes for a tool nothing actually targets — e.g. tuning
		// the Claude side of an agent would start requiring [models.opencode.*].
		if s.IsTuneOnly() {
			continue
		}
		for _, target := range s.TargetsOrAll() {
			seen[target] = true
		}
	}
	out := make([]string, 0, len(seen))
	for tool := range seen {
		out = append(out, tool)
	}
	sort.Strings(out)
	return out
}

// ModelSettingsScope reports where a tool's route-derived default-model
// settings belong: "project" when every model-backed resource enabled for the
// tool (builtin framework, command, or subagent — the same set
// EnabledModelTools counts) is project-scoped, so the models exist only to
// serve this repository; "user" otherwise — any user-scope resource, or no
// model-backed resource at all, keeps the global default-model projection.
// Both tools read a project-level settings file that overrides the global one
// (OpenCode: <repo>/opencode.jsonc; Claude: <repo>/.claude/settings.json), so
// a project-scoped workflow's models never leak into other projects' sessions.
func (c *Config) ModelSettingsScope(tool string) string {
	backed := false
	for _, r := range c.Frameworks {
		// Mirrors EnabledModelTools: only a builtin framework forces model
		// routing; a local skills-only framework does not.
		if !strings.HasPrefix(r.Source, "builtin:") {
			continue
		}
		if containsString(r.TargetsOrAll(), tool) {
			backed = true
			if r.Scope != "project" {
				return "user"
			}
		}
	}
	for _, r := range c.Commands {
		if containsString(r.TargetsOrAll(), tool) {
			backed = true
			if r.Scope != "project" {
				return "user"
			}
		}
	}
	for _, s := range c.Subagents {
		if s.IsTuneOnly() {
			continue
		}
		if containsString(s.TargetsOrAll(), tool) {
			backed = true
			if s.ScopeOrDefault() != "project" {
				return "user"
			}
		}
	}
	if !backed {
		return "user"
	}
	return "project"
}

func entriesForTool(resources map[string]Resource, tool string) []NamedResource {
	var out []NamedResource
	for name, r := range resources {
		if containsString(r.TargetsOrAll(), tool) {
			out = append(out, NamedResource{Name: name, Resource: r})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func containsString(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}

// Load reads and parses a homonto.toml file into a Config.
// decode parses the raw TOML and enforces the schema-version forward-safety
// guard. It is the first config-loading phase.
func decode(data []byte) (*Config, error) {
	var c Config
	if err := toml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	// Forward-safety: refuse a config from a newer schema before any adapter,
	// plan, or apply logic runs, so an older binary never silently mis-applies
	// fields it does not understand (TOML unmarshal drops unknown keys). Absent/0
	// is a legacy config, treated as the current version.
	if c.SchemaVersion > CurrentConfigSchemaVersion {
		return nil, fmt.Errorf("parse config: unknown config schema version %d (this binary supports up to %d) — upgrade homonto", c.SchemaVersion, CurrentConfigSchemaVersion)
	}
	return &c, nil
}

// migrate folds legacy declaration forms into their current equivalents.
func migrate(c *Config) {
	// Option C: the imperative [agents.<name>] model is superseded by the
	// declarative [subagents.<name>] one. Fold every declared agent into an
	// equivalent copy-mode subagent (a declared [agents.X] wins over an explicit
	// [subagents.X] of the same name) and drop the agents table, so [agents.X]
	// still parses but is now projected by `apply` like any other subagent.
	if len(c.Agents) > 0 {
		if c.Subagents == nil {
			c.Subagents = map[string]Subagent{}
		}
		for name, ag := range c.Agents {
			mode := ag.Mode
			if mode == "" && strings.HasPrefix(ag.Source, "builtin:") {
				mode = "copy" // builtin agents had no linkable path — copy-only
			}
			// [agents.X] wins the DECLARATION, but a same-named [subagents.X]'s
			// per-tool model blocks are tuning, which [agents.X] has no syntax
			// for — carry them over instead of silently deleting them.
			prev := c.Subagents[name]
			c.Subagents[name] = Subagent{
				Source:   ag.Source,
				Scope:    "user", // agents installed at user scope
				Mode:     mode,
				Version:  ag.Version,
				Targets:  ag.Targets,
				Claude:   prev.Claude,
				OpenCode: prev.OpenCode,
			}
		}
		c.Agents = nil
	}
}

// normalize applies defaulting so downstream projection sees concrete values.
func normalize(c *Config) {
	// Subagents default to project scope when omitted (skills and commands still
	// require an explicit scope). Normalize before validation so downstream
	// projection sees a concrete scope. Model-route values are whitespace-trimmed
	// here too: validation used to trim while the render did not, so
	// `model = "opus "` passed the alias check and then missed the alias map at
	// render, silently dropping its variant.
	trimRoute := func(r ModelRoute) ModelRoute {
		return ModelRoute{
			Model:   strings.TrimSpace(r.Model),
			Effort:  strings.TrimSpace(r.Effort),
			Variant: strings.TrimSpace(r.Variant),
		}
	}
	for name, r := range c.Subagents {
		if r.Scope == "" {
			r.Scope = "project"
		}
		r.Claude = trimRoute(r.Claude)
		r.OpenCode = trimRoute(r.OpenCode)
		c.Subagents[name] = r
	}
	for _, routes := range []map[string]ModelRoute{c.Models.Claude, c.Models.OpenCode} {
		for level, r := range routes {
			routes[level] = trimRoute(r)
		}
	}
}

// validate rejects a config that would project nothing or corrupt a tool file.
func validate(c *Config) error {
	for kind, resources := range map[string]map[string]Resource{
		"skills":   c.Skills,
		"commands": c.Commands,
	} {
		if err := validateResources(kind, resources); err != nil {
			return err
		}
	}
	// Frameworks have their own source rule: builtin:<name> (expanded from the
	// embedded catalog) or local:<path> (a local framework root). Every other
	// source expands nothing and is rejected loudly (F35).
	if err := validateFrameworkResources(c.Frameworks); err != nil {
		return err
	}
	// onto and to are an exclusive choice per repository: enterprise tooling
	// vs. simple development. Their skills give conflicting process guidance
	// and their binaries each expect to own the workflow, so declaring both
	// is a config error, not a projection concern.
	if _, hasOnto := c.Frameworks["onto"]; hasOnto {
		if _, hasTo := c.Frameworks["to"]; hasTo {
			return fmt.Errorf("parse config: [frameworks.onto] and [frameworks.to] are mutually exclusive; pick one workflow framework per repository (onto for evidence-gated enterprise changes, to for simple development)")
		}
	}
	if err := validateSubagents(c.Subagents); err != nil {
		return err
	}
	if err := validateModels(c); err != nil {
		return err
	}
	// Every other name becomes a key written into a tool's JSON file. sjson
	// treats index-like segments ("0", "-1") as array positions, silently
	// turning the containing object into a JSON ARRAY; empty names address
	// nothing. Reject both up front with the offending entry named.
	for name, m := range c.MCPs {
		if err := validateKey("mcps", name); err != nil {
			return err
		}
		// An MCP with no command cannot project — both adapters would skip it,
		// so a declared server would silently do nothing. Fail fast instead.
		if len(m.Command) == 0 {
			return fmt.Errorf("parse config: mcps entry %q has no command; an MCP server needs a command to run", name)
		}
		// A target that names no known tool matches no adapter, so the MCP is
		// projected nowhere — a silent typo. Only claude and opencode exist.
		for _, target := range m.Targets {
			if !isMCPTarget(target) {
				return fmt.Errorf("parse config: mcps entry %q targets unknown tool %q; valid targets are \"claude\", \"opencode\", and \"codex\"", name, target)
			}
		}
		switch m.Scope {
		case "", "user", "project":
			// ok
		default:
			return fmt.Errorf("parse config: mcps entry %q scope %q is invalid; valid values are \"user\" and \"project\"", name, m.Scope)
		}
		// Codex has no project-level config in the MCP pilot, so a
		// project-scoped server could only silently project globally there —
		// reject the combination instead.
		if m.ScopeOrDefault() == "project" && containsString(m.Targets, "codex") {
			return fmt.Errorf("parse config: mcps entry %q is project-scoped but targets codex, which supports only user scope (~/.codex/config.toml)", name)
		}
	}
	for _, tool := range []struct {
		name string
		m    map[string]Plugin
	}{
		{"plugins.claude", c.Plugins.Claude},
		{"plugins.opencode", c.Plugins.OpenCode},
	} {
		// Both adapters project keyed by source, so two decl names sharing a
		// source would collide on one projected key with last-writer-wins over
		// random map iteration order — a non-deterministic plan. Reject it.
		seenSource := map[string]string{} // source -> first decl name
		for declName, pl := range tool.m {
			if err := validateKey(tool.name, declName); err != nil {
				return err
			}
			// A plugin with no source projects nothing (no enabledPlugins key /
			// no plugin-array value), so a declared plugin would silently do
			// nothing. Fail fast naming the plugin.
			if strings.TrimSpace(pl.Source) == "" {
				return fmt.Errorf("parse config: %s plugin %q has an empty source", tool.name, declName)
			}
			// OpenCode plugins are a plain array on disk with no per-plugin
			// config slot, so a declared config could project nowhere. Reject it.
			if tool.name == "plugins.opencode" && len(pl.Config) > 0 {
				return fmt.Errorf("parse config: %s plugin %q declares config, but OpenCode has no per-plugin config on disk (its plugins are a plain array); remove config", tool.name, declName)
			}
			if prev, dup := seenSource[pl.Source]; dup {
				return fmt.Errorf("parse config: %s plugins %q and %q share source %q", tool.name, prev, declName, pl.Source)
			}
			seenSource[pl.Source] = declName
		}
	}
	// Marketplace declarations project to extraKnownMarketplaces.<name>. Each
	// source kind requires its locator field(s); an unknown source or a missing
	// locator projects nothing meaningful, so fail fast naming the marketplace.
	for name, mk := range c.Marketplaces.Claude {
		if err := validateKey("marketplaces.claude", name); err != nil {
			return err
		}
		switch mk.Source {
		case "github":
			if mk.Repo == "" {
				return fmt.Errorf("parse config: marketplaces.claude %q with source \"github\" is missing required \"repo\"", name)
			}
		case "url":
			if mk.URL == "" {
				return fmt.Errorf("parse config: marketplaces.claude %q with source \"url\" is missing required \"url\"", name)
			}
		case "git-subdir":
			if mk.URL == "" || mk.Path == "" {
				return fmt.Errorf("parse config: marketplaces.claude %q with source \"git-subdir\" is missing required \"url\" and/or \"path\"", name)
			}
		case "directory":
			if mk.Path == "" {
				return fmt.Errorf("parse config: marketplaces.claude %q with source \"directory\" is missing required \"path\"", name)
			}
		default:
			return fmt.Errorf("parse config: marketplaces.claude %q has unknown source %q; valid sources are \"github\", \"url\", \"git-subdir\", \"directory\"", name, mk.Source)
		}
	}
	// Settings keys that homonto itself manages in the same tool file would
	// collide with its own writes: claude projects plugins as `enabledPlugins`
	// into settings.json; opencode projects MCPs and plugins as the `mcp` and
	// `plugin` structures in opencode.jsonc. Reject those reserved names.
	//
	// `mcpServers` is reserved too: claude's current() deliberately skips that
	// settings.json key when reading managed values back (MCP servers are owned
	// via [mcps], projected into .claude.json). A settings.claude.mcpServers
	// value would be written on apply but never read back, so every plan would
	// re-propose it — a non-idempotent loop. Reject it up front instead.
	for k := range c.Settings.Claude {
		if err := validateKey("settings.claude", k); err != nil {
			return err
		}
		if k == "enabledPlugins" {
			return fmt.Errorf("parse config: settings.claude key %q is reserved (homonto manages plugins there); rename it", k)
		}
		if k == "mcpServers" {
			return fmt.Errorf("parse config: settings.claude key %q is reserved (homonto manages MCP servers via [mcps]); declare the server under [mcps] instead", k)
		}
		if k == "pluginConfigs" {
			return fmt.Errorf("parse config: settings.claude key %q is reserved (homonto manages pluginConfigs via [plugins.claude.<name>.config]); declare per-plugin config there instead", k)
		}
		if k == "extraKnownMarketplaces" {
			return fmt.Errorf("parse config: settings.claude key %q is reserved (homonto manages marketplaces via [marketplaces.claude.<name>]); declare the marketplace there instead", k)
		}
	}
	for k := range c.Settings.OpenCode {
		if err := validateKey("settings.opencode", k); err != nil {
			return err
		}
		if k == "mcp" || k == "plugin" {
			return fmt.Errorf("parse config: settings.opencode key %q is reserved (homonto manages %s there); rename it", k, k)
		}
	}
	// [tui.opencode] keys project into a second managed file (tui.json). Reject
	// index-like/empty names for the same JSON-array-corruption reason as
	// [settings.opencode].
	for k := range c.TUI.OpenCode {
		if err := validateKey("tui.opencode", k); err != nil {
			return err
		}
	}
	return nil
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	c, err := decode(data)
	if err != nil {
		return nil, err
	}
	migrate(c)
	normalize(c)
	if err := validate(c); err != nil {
		return nil, err
	}
	if abs, err := filepath.Abs(filepath.Dir(path)); err == nil {
		c.baseDir = abs
	} else {
		c.baseDir = filepath.Dir(path)
	}
	return c, nil
}

// validateKey rejects names unusable as literal JSON object keys: empty, or
// index-like (all digits, or "-" followed by digits — sjson array semantics).
func validateKey(kind, name string) error {
	if name == "" {
		return fmt.Errorf("parse config: %s entry %q is empty", kind, name)
	}
	if indexLike(name) {
		return fmt.Errorf("parse config: %s entry %q would be treated as a JSON array index and corrupt the target file; rename it", kind, name)
	}
	return nil
}

// validateResources checks name, scope, source, and targets for every declared
// resource of a given kind (frameworks, skills, commands, subagents).
func validateResources(kind string, resources map[string]Resource) error {
	for name, r := range resources {
		if err := validateResourceName(kind, name); err != nil {
			return err
		}
		label := kind + "." + name
		switch r.Scope {
		case "user", "project":
			// ok
		case "":
			return fmt.Errorf("parse config: %s is missing required scope; valid values are \"user\" and \"project\"", label)
		default:
			return fmt.Errorf("parse config: %s scope %q is invalid; valid values are \"user\" and \"project\"", label, r.Scope)
		}
		if err := validateSource(label, r.Source, r.Digest, false); err != nil {
			return err
		}
		if err := validateLocalPlainName(label, r.Source); err != nil {
			return err
		}
		for _, target := range r.Targets {
			if !isResourceTarget(target) {
				return fmt.Errorf("parse config: %s targets unknown tool %q; valid targets are \"claude\" and \"opencode\"", label, target)
			}
		}
	}
	return nil
}

// validateFrameworkResources validates [frameworks.X] entries. A framework
// source must be builtin:<name> (expanded from the embedded catalog),
// local:<path> (a local framework root resolved relative to the config dir), or
// remote:<url> (a framework root fetched through the trust pipeline, which
// REQUIRES a sha256 digest pin). Unlike skills/commands, a local FRAMEWORK
// source MAY carry path components, so the plain-name guard is deliberately not
// applied here. A bare name or a typo expands nothing and is rejected loudly
// (F35); a digest on a builtin/local source is a no-op and rejected.
func validateFrameworkResources(resources map[string]Resource) error {
	for name, r := range resources {
		if err := validateResourceName("frameworks", name); err != nil {
			return err
		}
		label := "frameworks." + name
		switch r.Scope {
		case "user", "project":
			// ok
		case "":
			return fmt.Errorf("parse config: %s is missing required scope; valid values are \"user\" and \"project\"", label)
		default:
			return fmt.Errorf("parse config: %s scope %q is invalid; valid values are \"user\" and \"project\"", label, r.Scope)
		}
		// A remote: framework installs through the same trust pipeline as a remote
		// subagent, so it REQUIRES a valid sha256 digest pin (parsed here so a
		// malformed remote framework fails at load, mirroring remote subagents).
		// builtin:/local: keep their existing rule: a digest on them is a no-op and
		// rejected.
		if remote.IsRemoteSource(r.Source) {
			if _, err := remote.ParseRemoteSource(r.Source); err != nil {
				return fmt.Errorf("parse config: %s %v", label, err)
			}
			if r.Digest == "" {
				return fmt.Errorf("parse config: %s remote source %q requires a digest = \"sha256:<hex>\" pin", label, r.Source)
			}
			if _, err := remote.ParseDigest(r.Digest); err != nil {
				return fmt.Errorf("parse config: %s %v", label, err)
			}
			for _, target := range r.Targets {
				if !isResourceTarget(target) {
					return fmt.Errorf("parse config: %s targets unknown tool %q; valid targets are \"claude\" and \"opencode\"", label, target)
				}
			}
			continue
		}
		if r.Digest != "" {
			return fmt.Errorf("parse config: %s digest is only valid on a remote: source", label)
		}
		builtinOK := strings.HasPrefix(r.Source, "builtin:") && strings.TrimPrefix(r.Source, "builtin:") != ""
		localOK := strings.HasPrefix(r.Source, "local:") && strings.TrimPrefix(r.Source, "local:") != ""
		if !builtinOK && !localOK {
			return fmt.Errorf("parse config: %s source %q must be a builtin:<name>, local:<path>, or remote:<url> source (another source would expand nothing)", label, r.Source)
		}
		for _, target := range r.Targets {
			if !isResourceTarget(target) {
				return fmt.Errorf("parse config: %s targets unknown tool %q; valid targets are \"claude\" and \"opencode\"", label, target)
			}
		}
	}
	return nil
}

// validateLocalPlainName rejects a local: source that is not a plain name (no
// `.`/`..`/path separators), so it can never resolve/link/materialize a file
// outside the provider root. It is a no-op for non-local sources. Shared by
// validateResources (skills/commands) and validateSubagents so the two paths
// cannot drift.
func validateLocalPlainName(label, source string) error {
	src, ok := strings.CutPrefix(source, "local:")
	if !ok {
		return nil
	}
	if src == "" || src == "." || src == ".." || strings.ContainsAny(src, `/\`) || src != filepath.Base(src) {
		return fmt.Errorf("parse config: %s local source %q must be a plain name (no path components)", label, source)
	}
	return nil
}

// validateSubagents checks each [subagents.<name>]: a valid name, a builtin/local
// source, known targets, a user|project scope (already normalized to project when
// omitted), and a mode of link. copy is reserved for the forthcoming copy-mode
// projection and rejected until that lands, so the field is never a silent no-op.
func validateSubagents(subagents map[string]Subagent) error {
	for name, s := range subagents {
		if err := validateResourceName("subagents", name); err != nil {
			return err
		}
		label := "subagents." + name
		// A tune-only entry ([subagents.<name>.<tool>] with no source) retunes an
		// agent a framework already declared, so the declaration rules — source,
		// scope, local-name safety — are not its to satisfy. Its model blocks are
		// still validated, by validateSubagentOverrides.
		if s.IsTuneOnly() {
			continue
		}
		switch s.Scope {
		case "user", "project":
			// ok (empty was normalized to project at load)
		default:
			return fmt.Errorf("parse config: %s scope %q is invalid; valid values are \"user\" and \"project\"", label, s.Scope)
		}
		if err := validateSource(label, s.Source, s.Digest, true); err != nil {
			return err
		}
		// A local: source is resolved to a file by name; reject a path-traversal
		// name so it cannot read/link outside the provider root.
		if err := validateLocalPlainName(label, s.Source); err != nil {
			return err
		}
		for _, target := range s.Targets {
			if !isResourceTarget(target) {
				return fmt.Errorf("parse config: %s targets unknown tool %q; valid targets are \"claude\" and \"opencode\"", label, target)
			}
		}
		switch s.Mode {
		case "", "link", "copy":
			// ok — link projects a symlink, copy projects a managed content file
		default:
			return fmt.Errorf("parse config: %s mode %q is invalid; valid values are \"link\" and \"copy\"", label, s.Mode)
		}
	}
	return nil
}

// mcpTargetTools are valid MCP targets. codex is a pilot adapter that projects
// MCP servers only, so it is a valid MCP target but NOT a valid target for
// skills/commands/subagents/frameworks (which it cannot project, and which would
// otherwise demand an unsatisfiable models.codex.* route via validateModels).
var mcpTargetTools = []string{"claude", "opencode", "codex"}
var resourceTargetTools = []string{"claude", "opencode"}

func isMCPTarget(t string) bool      { return slices.Contains(mcpTargetTools, t) }
func isResourceTarget(t string) bool { return slices.Contains(resourceTargetTools, t) }

func validateResourceName(kind, name string) error {
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\`) || name != filepath.Base(name) {
		return fmt.Errorf("parse config: %s entry %q is not a plain name", kind, name)
	}
	return validateKey(kind, name)
}

func validSource(source string) bool {
	for _, prefix := range []string{"builtin:", "local:"} {
		if strings.HasPrefix(source, prefix) && strings.TrimPrefix(source, prefix) != "" {
			return true
		}
	}
	return false
}

// validateSource accepts builtin:/local: sources unchanged, and a remote:
// source only when allowRemote is set (subagents only today), it parses, and it
// carries a well-formed sha256 digest pin. A non-remote source carrying a digest
// is rejected as unexpected so the field is never a silent no-op.
func validateSource(label, source, digest string, allowRemote bool) error {
	if remote.IsRemoteSource(source) {
		if !allowRemote {
			return fmt.Errorf("parse config: %s remote sources are only supported for subagents", label)
		}
		if _, err := remote.ParseRemoteSource(source); err != nil {
			return fmt.Errorf("parse config: %s %v", label, err)
		}
		if digest == "" {
			return fmt.Errorf("parse config: %s remote source %q requires a digest = \"sha256:<hex>\" pin", label, source)
		}
		if _, err := remote.ParseDigest(digest); err != nil {
			return fmt.Errorf("parse config: %s %v", label, err)
		}
		return nil
	}
	if digest != "" {
		return fmt.Errorf("parse config: %s digest is only valid on a remote: source", label)
	}
	if !validSource(source) {
		return fmt.Errorf("parse config: %s source %q is invalid; use builtin:<name>, local:<name>, or remote:<url>", label, source)
	}
	return nil
}

// The Claude effort/alias sets live in agentfm (the render is what actually
// speaks Claude's dialect); validation references the same maps so the two
// can never drift apart.
var (
	claudeEffortLevels = agentfm.ClaudeEffortLevels
	claudeModelAliases = agentfm.ClaudeAliases
)

// validateModelSpec checks one model/variant/effort triple against what `tool`
// can actually express, naming label as the offender. `model` is required only
// of a tier (a per-subagent override may set effort alone and inherit the rest),
// which requireModel selects.
//
// The tools differ, so the rules do:
//   - Claude renders a variant by bracketing an ALIAS (`opus[1m]`), and takes
//     `effort:` from a fixed set.
//   - OpenCode has a first-class `variant` field (any provider-defined string)
//     and no effort concept at all.
func validateModelSpec(tool, label string, r ModelRoute, requireModel bool) error {
	model := strings.TrimSpace(r.Model)
	variant := strings.TrimSpace(r.Variant)
	effort := strings.TrimSpace(r.Effort)
	if requireModel && model == "" {
		return fmt.Errorf("parse config: %s model is required", label)
	}
	switch tool {
	case "claude":
		if effort != "" && !claudeEffortLevels[effort] {
			return fmt.Errorf("parse config: %s effort %q is not a Claude effort level (low, medium, high, xhigh, max)", label, effort)
		}
		// Only meaningful against a model we can see; an override that sets a
		// variant alone is checked against the tier it merges into, below.
		if variant != "" && model != "" && !claudeModelAliases[model] {
			return fmt.Errorf("parse config: %s variant %q needs a model alias (opus, sonnet, haiku, fable, opusplan) — Claude takes no variant on the full model id %q", label, variant, model)
		}
	case "opencode":
		if effort != "" {
			return fmt.Errorf("parse config: %s sets effort %q, but OpenCode has no effort setting — use variant, or drop it", label, effort)
		}
	}
	return nil
}

// validateModels ensures every tool enabled by a non-skill resource declares all
// four model tiers with a model, and that every model/variant/effort value —
// tier or per-subagent override — is one the target tool can actually express.
//
// Effort and variant are OPTIONAL: a tier naming just a model is complete. They
// were once mandatory while being projected nowhere, which meant homonto forced
// you to write a field it then discarded — and never checked, so configs filled
// with values no tool accepts.
func validateModels(c *Config) error {
	// An unknown tier name ([models.opencode.reviewing], say) matches no agent
	// role and no default-model projection, so it would validate clean and then
	// do nothing — reject it naming the offender. agentfm.TierNames is the
	// single source of truth the role check in rendering uses too.
	for tool, routes := range map[string]map[string]ModelRoute{
		"claude":   c.Models.Claude,
		"opencode": c.Models.OpenCode,
	} {
		levels := make([]string, 0, len(routes))
		for level := range routes {
			levels = append(levels, level)
		}
		sort.Strings(levels) // deterministic: the same config must fail on the same offender
		for _, level := range levels {
			if !agentfm.Tiers[level] {
				return fmt.Errorf("parse config: models.%s.%s is not a model tier; valid tiers are %q, %q, %q, %q (agents pick one via their role)", tool, level, agentfm.TierNames[0], agentfm.TierNames[1], agentfm.TierNames[2], agentfm.TierNames[3])
			}
		}
	}
	for _, tool := range c.EnabledModelTools() {
		for _, level := range agentfm.TierNames {
			route, ok := modelRouteFor(c.Models, tool, level)
			label := "models." + tool + "." + level
			if !ok {
				return fmt.Errorf("parse config: %s is required for enabled target tool %q", label, tool)
			}
			if err := validateModelSpec(tool, label, route, true); err != nil {
				return err
			}
		}
	}
	return validateSubagentOverrides(c)
}

// validateSubagentOverrides checks every [subagents.<name>.<tool>] block —
// deliberately IGNORING the entry's targets, because the engine applies
// overrides unconditionally when it renders both tools' variants. The previous
// version iterated TargetsOrAll(), which let an untargeted tool's block skip
// validation entirely and stamp any value straight into a live agent file.
//
// It also rejects the two silent-no-op classes the review found: an override on
// a local:/remote: source (that content is never rendered, so the override can
// never apply), and a tune-only entry naming an agent that is not installed (a
// typo'd name would otherwise validate, plan, and apply clean while retuning
// nothing).
func validateSubagentOverrides(c *Config) error {
	names := make([]string, 0, len(c.Subagents))
	for name := range c.Subagents {
		names = append(names, name)
	}
	sort.Strings(names) // deterministic: the same config must fail on the same offender

	// The installed builtin agents, by catalog name — what a tune-only entry
	// must resolve against. Computed lazily: only configs that carry overrides
	// pay for the framework expansion.
	var installed map[string]bool
	installedBuiltins := func() (map[string]bool, error) {
		if installed != nil {
			return installed, nil
		}
		installed = map[string]bool{}
		for _, tool := range []string{"claude", "opencode"} {
			entries, err := c.ExpandedSubagentEntriesForTool(tool)
			if err != nil {
				return nil, err
			}
			for _, e := range entries {
				if cat, ok := SubagentCatalogName(e.Resource.Source); ok {
					installed[cat] = true
				}
			}
		}
		return installed, nil
	}

	seen := map[string]map[string]struct {
		entry string
		ov    ModelRoute
	}{} // catalog name -> tool -> first override seen
	for _, name := range names {
		sa := c.Subagents[name]
		hasOverride := sa.Claude != (ModelRoute{}) || sa.OpenCode != (ModelRoute{})
		if !hasOverride {
			continue
		}

		// Resolve the catalog name the override applies to. Overrides only make
		// sense for builtin (catalog-rendered) agents: local:/remote: content is
		// projected verbatim, so an override there would be accepted and then
		// silently discarded — reject it instead.
		cat := name
		if !sa.IsTuneOnly() {
			var ok bool
			if cat, ok = SubagentCatalogName(sa.Source); !ok {
				return fmt.Errorf("parse config: subagents.%s declares a model override, but its source %q is not builtin: — local:/remote: agents are projected verbatim and never rendered, so the override could never apply", name, sa.Source)
			}
		} else {
			known, err := installedBuiltins()
			if err != nil {
				return err
			}
			if !known[cat] {
				return fmt.Errorf("parse config: subagents.%s tunes an agent that is not installed — no framework or [subagents.*] declaration provides builtin:%s (typo?)", name, cat)
			}
		}

		for _, tool := range []string{"claude", "opencode"} {
			ov := sa.ModelOverrideFor(tool)
			if ov == (ModelRoute{}) {
				continue
			}
			label := "subagents." + name + "." + tool
			// Validate the fragment itself. A variant whose model comes from the
			// tier cannot be judged here — which tier depends on the agent's
			// frontmatter role, known only at render — so agentfm.Render errors
			// loudly on an unrenderable merged combination instead.
			if err := validateModelSpec(tool, label, ov, false); err != nil {
				return err
			}
			// Conflicts are judged per CATALOG name: one builtin renders one
			// file, so two entries' overrides for it must agree or the winner
			// would be map-iteration luck (a different render — and a different
			// materialize fingerprint — every run).
			if seen[cat] == nil {
				seen[cat] = map[string]struct {
					entry string
					ov    ModelRoute
				}{}
			}
			if prev, dup := seen[cat][tool]; dup && prev.ov != ov {
				return fmt.Errorf("parse config: subagents.%s.%s conflicts with subagents.%s.%s — one builtin (%s) renders one file, so its overrides must agree", name, tool, prev.entry, tool, cat)
			}
			seen[cat][tool] = struct {
				entry string
				ov    ModelRoute
			}{name, ov}
		}
	}
	return nil
}

func modelRouteFor(models ModelConfig, tool, level string) (ModelRoute, bool) {
	switch tool {
	case "claude":
		r, ok := models.Claude[level]
		return r, ok
	case "opencode":
		r, ok := models.OpenCode[level]
		return r, ok
	default:
		return ModelRoute{}, false
	}
}

// indexLike reports whether sjson would treat name as an array index:
// all-digit ("0", "42") or "-" followed by digits ("-1", the append form).
func indexLike(name string) bool {
	t := strings.TrimPrefix(name, "-")
	if t == "" {
		return false // "-" alone is a plain key
	}
	for i := 0; i < len(t); i++ {
		if t[i] < '0' || t[i] > '9' {
			return false
		}
	}
	return true
}
