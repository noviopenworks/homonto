package config

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"

	cat "github.com/noviopenworks/homonto/internal/catalog"
	toml "github.com/pelletier/go-toml/v2"
)

// MCP is a declared MCP server. Env values may hold unresolved ${...} tokens.
type MCP struct {
	Command []string          `toml:"command"`
	Env     map[string]string `toml:"env"`
	Targets []string          `toml:"targets"`
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

// Config is the tool-agnostic desired state parsed from homonto.toml.
type Config struct {
	MCPs         map[string]MCP      `toml:"mcps"`
	Frameworks   map[string]Resource `toml:"frameworks"`
	Skills       map[string]Resource `toml:"skills"`
	Commands     map[string]Resource `toml:"commands"`
	Subagents    map[string]Resource `toml:"subagents"`
	Models       ModelConfig         `toml:"models"`
	Plugins      Plugins             `toml:"plugins"`
	Settings     Settings            `toml:"settings"`
	TUI          TUI                 `toml:"tui"`
	Marketplaces Marketplaces        `toml:"marketplaces"`
	Agents       map[string]Agent    `toml:"agents"`
}

func (c *Config) SkillEntriesForTool(tool string) []NamedResource {
	return entriesForTool(c.Skills, tool)
}

func (c *Config) CommandEntriesForTool(tool string) []NamedResource {
	return entriesForTool(c.Commands, tool)
}

func (c *Config) SubagentEntriesForTool(tool string) []NamedResource {
	return entriesForTool(c.Subagents, tool)
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

func sameResource(a, b Resource) bool {
	return a.Source == b.Source && a.Scope == b.Scope && slices.Equal(a.Targets, b.Targets)
}

// ExpandedSkillEntriesForTool returns the effective skills for a tool: explicit
// [skills.X] entries plus, for each [frameworks.<fw>] source="builtin:<fw>"
// targeting the tool, its transitively expanded skills. Each expanded skill
// inherits the framework declaration's scope and targets. A framework skill
// whose name collides with an explicit [skills.X] entry, or with another
// framework's skill under a conflicting declaration, is an error, as is a
// dependency cycle (surfaced from catalog.Expand).
func (c *Config) ExpandedSkillEntriesForTool(tool string) ([]NamedResource, error) {
	byName := map[string]NamedResource{}
	explicitNames := map[string]bool{}
	for _, e := range c.SkillEntriesForTool(tool) {
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
		if !strings.HasPrefix(fwRes.Source, "builtin:") {
			continue
		}
		if !containsString(fwRes.TargetsOrAll(), tool) {
			continue
		}
		if cl == nil {
			var err error
			if cl, err = loadedCatalog(); err != nil {
				return nil, err
			}
		}
		builtin := strings.TrimPrefix(fwRes.Source, "builtin:")
		expanded, err := cl.Expand([]string{builtin})
		if err != nil {
			return nil, fmt.Errorf("config: framework %q: %w", fwName, err)
		}
		for _, es := range expanded {
			if explicitNames[es.Name] {
				return nil, fmt.Errorf("config: skill %q is declared both explicitly in [skills] and by framework %q", es.Name, fwName)
			}
			nr := NamedResource{
				Name: es.Name,
				Resource: Resource{
					Source:  "builtin:" + es.Name,
					Scope:   fwRes.Scope,
					Targets: fwRes.Targets,
				},
			}
			if prev, ok := byName[es.Name]; ok {
				if !sameResource(prev.Resource, nr.Resource) {
					return nil, fmt.Errorf("config: skill %q expanded by multiple frameworks with conflicting scope/targets (framework %q)", es.Name, fwName)
				}
				continue
			}
			byName[es.Name] = nr
		}
	}

	out := make([]NamedResource, 0, len(byName))
	for _, nr := range byName {
		out = append(out, nr)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
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
	byName := map[string]NamedResource{}
	explicitNames := map[string]bool{}
	for _, e := range c.CommandEntriesForTool(tool) {
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
		if !strings.HasPrefix(fwRes.Source, "builtin:") {
			continue
		}
		if !containsString(fwRes.TargetsOrAll(), tool) {
			continue
		}
		if cl == nil {
			var err error
			if cl, err = loadedCatalog(); err != nil {
				return nil, err
			}
		}
		builtin := strings.TrimPrefix(fwRes.Source, "builtin:")
		expanded, err := cl.ExpandCommands([]string{builtin})
		if err != nil {
			return nil, fmt.Errorf("config: framework %q: %w", fwName, err)
		}
		for _, ec := range expanded {
			if explicitNames[ec.Name] {
				return nil, fmt.Errorf("config: command %q is declared both explicitly in [commands] and by framework %q", ec.Name, fwName)
			}
			nr := NamedResource{
				Name: ec.Name,
				Resource: Resource{
					Source:  "builtin:" + ec.Name,
					Scope:   fwRes.Scope,
					Targets: fwRes.Targets,
				},
			}
			if prev, ok := byName[ec.Name]; ok {
				if !sameResource(prev.Resource, nr.Resource) {
					return nil, fmt.Errorf("config: command %q expanded by multiple frameworks with conflicting scope/targets (framework %q)", ec.Name, fwName)
				}
				continue
			}
			byName[ec.Name] = nr
		}
	}

	out := make([]NamedResource, 0, len(byName))
	for _, nr := range byName {
		out = append(out, nr)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
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
	byName := map[string]NamedResource{}
	explicitNames := map[string]bool{}
	for _, e := range c.SubagentEntriesForTool(tool) {
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
		if !strings.HasPrefix(fwRes.Source, "builtin:") {
			continue
		}
		if !containsString(fwRes.TargetsOrAll(), tool) {
			continue
		}
		if cl == nil {
			var err error
			if cl, err = loadedCatalog(); err != nil {
				return nil, err
			}
		}
		builtin := strings.TrimPrefix(fwRes.Source, "builtin:")
		expanded, err := cl.ExpandSubagents([]string{builtin})
		if err != nil {
			return nil, fmt.Errorf("config: framework %q: %w", fwName, err)
		}
		for _, es := range expanded {
			if explicitNames[es.Name] {
				return nil, fmt.Errorf("config: subagent %q is declared both explicitly in [subagents] and by framework %q", es.Name, fwName)
			}
			nr := NamedResource{
				Name: es.Name,
				Resource: Resource{
					Source:  "builtin:" + es.Name,
					Scope:   fwRes.Scope,
					Targets: fwRes.Targets,
				},
			}
			if prev, ok := byName[es.Name]; ok {
				if !sameResource(prev.Resource, nr.Resource) {
					return nil, fmt.Errorf("config: subagent %q expanded by multiple frameworks with conflicting scope/targets (framework %q)", es.Name, fwName)
				}
				continue
			}
			byName[es.Name] = nr
		}
	}

	out := make([]NamedResource, 0, len(byName))
	for _, nr := range byName {
		out = append(out, nr)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (c *Config) EnabledModelTools() []string {
	seen := map[string]bool{}
	for _, resources := range []map[string]Resource{c.Frameworks, c.Commands, c.Subagents} {
		for _, r := range resources {
			for _, target := range r.TargetsOrAll() {
				seen[target] = true
			}
		}
	}
	out := make([]string, 0, len(seen))
	for tool := range seen {
		out = append(out, tool)
	}
	sort.Strings(out)
	return out
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
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var c Config
	if err := toml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	for kind, resources := range map[string]map[string]Resource{
		"frameworks": c.Frameworks,
		"skills":     c.Skills,
		"commands":   c.Commands,
		"subagents":  c.Subagents,
	} {
		if err := validateResources(kind, resources); err != nil {
			return nil, err
		}
	}
	if err := validateModels(&c); err != nil {
		return nil, err
	}
	if err := validateAgents(c.Agents); err != nil {
		return nil, err
	}
	// Every other name becomes a key written into a tool's JSON file. sjson
	// treats index-like segments ("0", "-1") as array positions, silently
	// turning the containing object into a JSON ARRAY; empty names address
	// nothing. Reject both up front with the offending entry named.
	for name, m := range c.MCPs {
		if err := validateKey("mcps", name); err != nil {
			return nil, err
		}
		// An MCP with no command cannot project — both adapters would skip it,
		// so a declared server would silently do nothing. Fail fast instead.
		if len(m.Command) == 0 {
			return nil, fmt.Errorf("parse config: mcps entry %q has no command; an MCP server needs a command to run", name)
		}
		// A target that names no known tool matches no adapter, so the MCP is
		// projected nowhere — a silent typo. Only claude and opencode exist.
		for _, target := range m.Targets {
			if target != "claude" && target != "opencode" {
				return nil, fmt.Errorf("parse config: mcps entry %q targets unknown tool %q; valid targets are \"claude\" and \"opencode\"", name, target)
			}
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
				return nil, err
			}
			// A plugin with no source projects nothing (no enabledPlugins key /
			// no plugin-array value), so a declared plugin would silently do
			// nothing. Fail fast naming the plugin.
			if strings.TrimSpace(pl.Source) == "" {
				return nil, fmt.Errorf("parse config: %s plugin %q has an empty source", tool.name, declName)
			}
			// OpenCode plugins are a plain array on disk with no per-plugin
			// config slot, so a declared config could project nowhere. Reject it.
			if tool.name == "plugins.opencode" && len(pl.Config) > 0 {
				return nil, fmt.Errorf("parse config: %s plugin %q declares config, but OpenCode has no per-plugin config on disk (its plugins are a plain array); remove config", tool.name, declName)
			}
			if prev, dup := seenSource[pl.Source]; dup {
				return nil, fmt.Errorf("parse config: %s plugins %q and %q share source %q", tool.name, prev, declName, pl.Source)
			}
			seenSource[pl.Source] = declName
		}
	}
	// Marketplace declarations project to extraKnownMarketplaces.<name>. Each
	// source kind requires its locator field(s); an unknown source or a missing
	// locator projects nothing meaningful, so fail fast naming the marketplace.
	for name, mk := range c.Marketplaces.Claude {
		if err := validateKey("marketplaces.claude", name); err != nil {
			return nil, err
		}
		switch mk.Source {
		case "github":
			if mk.Repo == "" {
				return nil, fmt.Errorf("parse config: marketplaces.claude %q with source \"github\" is missing required \"repo\"", name)
			}
		case "url":
			if mk.URL == "" {
				return nil, fmt.Errorf("parse config: marketplaces.claude %q with source \"url\" is missing required \"url\"", name)
			}
		case "git-subdir":
			if mk.URL == "" || mk.Path == "" {
				return nil, fmt.Errorf("parse config: marketplaces.claude %q with source \"git-subdir\" is missing required \"url\" and/or \"path\"", name)
			}
		case "directory":
			if mk.Path == "" {
				return nil, fmt.Errorf("parse config: marketplaces.claude %q with source \"directory\" is missing required \"path\"", name)
			}
		default:
			return nil, fmt.Errorf("parse config: marketplaces.claude %q has unknown source %q; valid sources are \"github\", \"url\", \"git-subdir\", \"directory\"", name, mk.Source)
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
			return nil, err
		}
		if k == "enabledPlugins" {
			return nil, fmt.Errorf("parse config: settings.claude key %q is reserved (homonto manages plugins there); rename it", k)
		}
		if k == "mcpServers" {
			return nil, fmt.Errorf("parse config: settings.claude key %q is reserved (homonto manages MCP servers via [mcps]); declare the server under [mcps] instead", k)
		}
		if k == "pluginConfigs" {
			return nil, fmt.Errorf("parse config: settings.claude key %q is reserved (homonto manages pluginConfigs via [plugins.claude.<name>.config]); declare per-plugin config there instead", k)
		}
		if k == "extraKnownMarketplaces" {
			return nil, fmt.Errorf("parse config: settings.claude key %q is reserved (homonto manages marketplaces via [marketplaces.claude.<name>]); declare the marketplace there instead", k)
		}
	}
	for k := range c.Settings.OpenCode {
		if err := validateKey("settings.opencode", k); err != nil {
			return nil, err
		}
		if k == "mcp" || k == "plugin" {
			return nil, fmt.Errorf("parse config: settings.opencode key %q is reserved (homonto manages %s there); rename it", k, k)
		}
	}
	// [tui.opencode] keys project into a second managed file (tui.json). Reject
	// index-like/empty names for the same JSON-array-corruption reason as
	// [settings.opencode].
	for k := range c.TUI.OpenCode {
		if err := validateKey("tui.opencode", k); err != nil {
			return nil, err
		}
	}
	return &c, nil
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
		if !validSource(r.Source) {
			return fmt.Errorf("parse config: %s source %q is invalid; use builtin:<name> or local:<name>", label, r.Source)
		}
		for _, target := range r.Targets {
			if target != "claude" && target != "opencode" {
				return fmt.Errorf("parse config: %s targets unknown tool %q; valid targets are \"claude\" and \"opencode\"", label, target)
			}
		}
	}
	return nil
}

// validateAgents checks name, source, mode, and targets for every declared
// [agents.<name>] lifecycle agent.
func validateAgents(agents map[string]Agent) error {
	for name, ag := range agents {
		if err := validateKey("agents", name); err != nil {
			return err
		}
		label := "agents." + name
		if !validSource(ag.Source) {
			return fmt.Errorf("parse config: %s source %q is invalid; use builtin:<name> or local:<name>", label, ag.Source)
		}
		switch ag.Mode {
		case "", "copy", "link":
		default:
			return fmt.Errorf("parse config: %s mode %q is invalid; valid values are \"copy\" and \"link\"", label, ag.Mode)
		}
		for _, target := range ag.Targets {
			if target != "claude" && target != "opencode" {
				return fmt.Errorf("parse config: %s targets unknown tool %q; valid targets are \"claude\" and \"opencode\"", label, target)
			}
		}
	}
	return nil
}

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

// validateModels ensures every tool enabled by a non-skill resource has all
// three model levels (architectural, coding, trivial) populated with a model
// and either an effort or variant.
func validateModels(c *Config) error {
	for _, tool := range c.EnabledModelTools() {
		for _, level := range []string{"architectural", "coding", "trivial"} {
			route, ok := modelRouteFor(c.Models, tool, level)
			label := "models." + tool + "." + level
			if !ok {
				return fmt.Errorf("parse config: %s is required for enabled target tool %q", label, tool)
			}
			if strings.TrimSpace(route.Model) == "" {
				return fmt.Errorf("parse config: %s model is required", label)
			}
			if strings.TrimSpace(route.Effort) == "" && strings.TrimSpace(route.Variant) == "" {
				return fmt.Errorf("parse config: %s requires effort or variant", label)
			}
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
