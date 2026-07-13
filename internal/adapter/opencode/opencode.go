package opencode

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/noviopenworks/homonto/internal/adapter"
	"github.com/noviopenworks/homonto/internal/adapter/copyproj"
	"github.com/noviopenworks/homonto/internal/adapter/fileproj"
	"github.com/noviopenworks/homonto/internal/adapter/jsoncodec"
	"github.com/noviopenworks/homonto/internal/adapter/structproj"
	"github.com/noviopenworks/homonto/internal/commandpath"
	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/copyfile"
	"github.com/noviopenworks/homonto/internal/fsutil"
	"github.com/noviopenworks/homonto/internal/jsonutil"
	"github.com/noviopenworks/homonto/internal/secret"
	"github.com/noviopenworks/homonto/internal/skillpath"
	"github.com/noviopenworks/homonto/internal/state"
	"github.com/noviopenworks/homonto/internal/subagentpath"
)

// Adapter projects desired config into OpenCode's opencode.jsonc under home.
type Adapter struct {
	home                string
	content             string
	catalogRoot         string // materialized builtin catalog root (.homonto/catalog/skills)
	commandCatalogRoot  string // materialized builtin command root (.homonto/catalog/commands)
	subagentCatalogRoot string // materialized builtin subagent root (.homonto/catalog/subagents)
	remoteSubagentRoot  string // materialized remote subagent root (.homonto/remote/subagents)
	projectRoot         string // directory of homonto.toml; used for project-scope resources
	skills              []config.NamedResource
	commands            []config.NamedResource
	subagents           []config.NamedResource
}

// New builds an OpenCode adapter at user scope. home is $HOME; content holds
// owned skills. Use WithProjectRoot to install project-scope skills.
func New(home, content string) *Adapter { return &Adapter{home: home, content: content} }

// WithProjectRoot sets the project root (the homonto.toml directory). It is
// used for project-scope resource placement. MCP servers, settings, and
// plugins always project under home.
func (a *Adapter) WithProjectRoot(projectRoot string) *Adapter {
	a.projectRoot = projectRoot
	return a
}

// WithCatalogRoot sets the materialized builtin-catalog root that builtin:<name>
// skills link from. Mirrors WithProjectRoot.
func (a *Adapter) WithCatalogRoot(catalogRoot string) *Adapter {
	a.catalogRoot = catalogRoot
	return a
}

// WithCommandCatalogRoot sets the materialized builtin-command root that
// builtin:<name> commands link from. Mirrors WithCatalogRoot.
func (a *Adapter) WithCommandCatalogRoot(commandCatalogRoot string) *Adapter {
	a.commandCatalogRoot = commandCatalogRoot
	return a
}

// WithSubagentCatalogRoot sets the materialized builtin-subagent root that
// builtin:<name> subagents link from. Mirrors WithCommandCatalogRoot.
func (a *Adapter) WithSubagentCatalogRoot(subagentCatalogRoot string) *Adapter {
	a.subagentCatalogRoot = subagentCatalogRoot
	return a
}

// WithRemoteSubagentRoot sets the materialized remote-subagent root that
// remote:<url> subagents link from (populated by the engine's verify pipeline
// before apply). Mirrors WithSubagentCatalogRoot.
func (a *Adapter) WithRemoteSubagentRoot(remoteSubagentRoot string) *Adapter {
	a.remoteSubagentRoot = remoteSubagentRoot
	return a
}

// managedRoots returns every content root homonto owns links into. catalogRoot,
// commandCatalogRoot, and subagentCatalogRoot are included only when set:
// link.managed() treats an empty-string root as a prefix match for every
// absolute path, so passing "" here would make link calls treat any symlink as
// "ours" — an empty root must never reach link.*.
func (a *Adapter) managedRoots() []string {
	roots := []string{a.content}
	if a.catalogRoot != "" {
		roots = append(roots, a.catalogRoot)
	}
	if a.commandCatalogRoot != "" {
		roots = append(roots, a.commandCatalogRoot)
	}
	if a.subagentCatalogRoot != "" {
		roots = append(roots, a.subagentCatalogRoot)
	}
	if a.remoteSubagentRoot != "" {
		roots = append(roots, a.remoteSubagentRoot)
	}
	return roots
}

// skillSource resolves a skill entry's on-disk content directory by source
// scheme: builtin:<n> from the materialized catalog root, otherwise the local
// content dir.
func (a *Adapter) skillSource(entry config.NamedResource) string {
	if s := entry.Resource.Source; strings.HasPrefix(s, "builtin:") {
		return filepath.Join(a.catalogRoot, strings.TrimPrefix(s, "builtin:"))
	}
	return filepath.Join(a.content, "skills", localSourceName(entry.Resource.Source, entry.Name))
}

func (a *Adapter) Name() string { return "opencode" }

func (a *Adapter) cfgFile() string {
	return filepath.Join(a.home, ".config", "opencode", "opencode.jsonc")
}

// tuiFile is the second managed file: OpenCode reads TUI settings from a
// separate ~/.config/opencode/tui.json. [tui.opencode] keys project here under
// the "tui." state namespace, independent of opencode.jsonc.
func (a *Adapter) tuiFile() string {
	return filepath.Join(a.home, ".config", "opencode", "tui.json")
}

// skillsDir is the directory owned-skill symlinks live in for the given scope.
func (a *Adapter) skillsDir(scope string) string {
	return skillpath.Dir("opencode", scope, a.home, a.projectRoot)
}

// inactiveSkillsDir is the other scope's skills directory — where a link may
// linger after a per-resource scope switch. It returns "" when there is nothing
// meaningful to relocate from: no project root is known, or the two scopes
// resolve to the same directory.
func (a *Adapter) inactiveSkillsDir(scope string) string {
	if a.projectRoot == "" {
		return ""
	}
	d := skillpath.Dir("opencode", skillpath.Other(scope), a.home, a.projectRoot)
	if d == a.skillsDir(scope) {
		return ""
	}
	return d
}

// commandsDir is the directory owned-command symlinks live in for the scope.
func (a *Adapter) commandsDir(scope string) string {
	return commandpath.Dir("opencode", scope, a.home, a.projectRoot)
}

// inactiveCommandsDir is the other scope's commands directory — where a link
// may linger after a per-resource scope switch. It returns "" when nothing
// meaningful can be relocated (no project root, or both scopes resolve equal).
func (a *Adapter) inactiveCommandsDir(scope string) string {
	if a.projectRoot == "" {
		return ""
	}
	d := commandpath.Dir("opencode", skillpath.Other(scope), a.home, a.projectRoot)
	if d == a.commandsDir(scope) {
		return ""
	}
	return d
}

// commandSource resolves a command entry's on-disk file by source scheme:
// builtin:<n> from the materialized command root (<n>.md), otherwise the local
// content dir (homonto/commands/<n>.md).
func (a *Adapter) commandSource(entry config.NamedResource) string {
	if s := entry.Resource.Source; strings.HasPrefix(s, "builtin:") {
		return filepath.Join(a.commandCatalogRoot, strings.TrimPrefix(s, "builtin:")+".md")
	}
	return filepath.Join(a.content, "commands", localSourceName(entry.Resource.Source, entry.Name)+".md")
}

// commandFileLinks builds the desired managed command symlinks for the fileproj
// contract (destination is <name>.md).
func (a *Adapter) commandFileLinks() []fileproj.Link {
	var out []fileproj.Link
	for _, e := range a.commands {
		inact := ""
		if d := a.inactiveCommandsDir(e.Resource.Scope); d != "" {
			inact = filepath.Join(d, e.Name+".md")
		}
		out = append(out, fileproj.Link{
			Dst:      filepath.Join(a.commandsDir(e.Resource.Scope), e.Name+".md"),
			Src:      a.commandSource(e),
			Key:      "command." + e.Name,
			Inactive: inact,
		})
	}
	return out
}

// subagentsDir is the directory owned-subagent symlinks live in for the scope.
func (a *Adapter) subagentsDir(scope string) string {
	return subagentpath.Dir("opencode", scope, a.home, a.projectRoot)
}

// inactiveSubagentsDir is the other scope's subagent directory — where a link
// may linger after a per-resource scope switch. It returns "" when nothing
// meaningful can be relocated (no project root, or both scopes resolve equal).
func (a *Adapter) inactiveSubagentsDir(scope string) string {
	if a.projectRoot == "" {
		return ""
	}
	d := subagentpath.Dir("opencode", skillpath.Other(scope), a.home, a.projectRoot)
	if d == a.subagentsDir(scope) {
		return ""
	}
	return d
}

// subagentSource resolves a subagent entry's on-disk file by source scheme:
// builtin:<n> from the materialized subagent root (<n>.md), remote:<url> from the
// materialized remote root (<name>.md), otherwise the local content dir
// (homonto/subagents/<n>.md).
func (a *Adapter) subagentSource(entry config.NamedResource) string {
	if s := entry.Resource.Source; strings.HasPrefix(s, "builtin:") {
		return filepath.Join(a.subagentCatalogRoot, strings.TrimPrefix(s, "builtin:")+".md")
	}
	if strings.HasPrefix(entry.Resource.Source, "remote:") {
		return filepath.Join(a.remoteSubagentRoot, entry.Name+".md")
	}
	return filepath.Join(a.content, "subagents", localSourceName(entry.Resource.Source, entry.Name)+".md")
}

// subagentFileLinks builds the desired managed subagent symlinks for the
// fileproj contract. Copy-mode subagents are projected as content files (not
// links), so they are skipped here and reconciled by applyCopySubagents.
func (a *Adapter) subagentFileLinks() []fileproj.Link {
	var out []fileproj.Link
	for _, e := range a.subagents {
		if e.Mode == "copy" {
			continue
		}
		inact := ""
		if d := a.inactiveSubagentsDir(e.Resource.Scope); d != "" {
			inact = filepath.Join(d, e.Name+".md")
		}
		out = append(out, fileproj.Link{
			Dst:      filepath.Join(a.subagentsDir(e.Resource.Scope), e.Name+".md"),
			Src:      a.subagentSource(e),
			Key:      "subagent." + e.Name,
			Inactive: inact,
		})
	}
	return out
}

// copySubagentDesired returns dst -> resolved content for each copy-mode subagent.
func (a *Adapter) copySubagentDesired() (map[string][]byte, error) {
	out := map[string][]byte{}
	for _, entry := range a.subagents {
		if entry.Mode != "copy" {
			continue
		}
		content, err := os.ReadFile(a.subagentSource(entry))
		if err != nil {
			return nil, err
		}
		out[filepath.Join(a.subagentsDir(entry.Resource.Scope), entry.Name+".md")] = content
	}
	return out, nil
}

// planCopyOps computes the reconciler ops for copy-mode subagents against state
// through the shared copyproj core.
func (a *Adapter) planCopyOps(st *state.State) ([]copyfile.Op, error) {
	desired, err := a.copySubagentDesired()
	if err != nil {
		return nil, err
	}
	return copyproj.Plan("opencode", desired, st)
}

// applyCopySubagents reconciles copy-mode subagent content files through the
// shared copyproj core (write/update/prune + local-edit .bak backup + state,
// conflict abort, F7 prune-root guard).
func (a *Adapter) applyCopySubagents(st *state.State) error {
	desired, err := a.copySubagentDesired()
	if err != nil {
		return err
	}
	return copyproj.Apply("opencode", desired, st, a.copyPruneRoots())
}

// copyPruneRoots are the directories a copy-mode subagent file may legitimately
// live in (user + project agent dirs). copyfile.Apply refuses to delete a prune
// destination — reconstructed from an untrusted state entry — that resolves
// outside these roots, so a tampered state.json path cannot delete an arbitrary
// file (F7). The project dir is included only when a project root is known.
func (a *Adapter) copyPruneRoots() []string {
	roots := []string{a.subagentsDir("user")}
	if a.projectRoot != "" {
		roots = append(roots, a.subagentsDir("project"))
	}
	return roots
}

// localSourceName resolves a skill resource's content subdirectory: a local:
// source names that directory directly; any other source falls back to the
// skill's declared name.
func localSourceName(source, fallback string) string {
	if strings.HasPrefix(source, "local:") {
		return strings.TrimPrefix(source, "local:")
	}
	return fallback
}

func (a *Adapter) desiredMCPs(c *config.Config) map[string]string {
	out := map[string]string{}
	for name, m := range c.MCPs {
		if !contains(m.TargetsOrAll(), "opencode") {
			continue
		}
		// No command means nothing runnable to project (matches claude's
		// adapter); writing `command: []` would just break the tool.
		if len(m.Command) == 0 {
			continue
		}
		obj := map[string]any{"type": "local", "command": m.Command, "enabled": true}
		if len(m.Env) > 0 {
			obj["environment"] = m.Env
		}
		out["mcp."+name] = mustJSON(obj)
	}
	return out
}

// desiredSettings maps each [settings.opencode] key to its setting.* state key.
func desiredSettings(c *config.Config) map[string]string {
	out := map[string]string{}
	for k, v := range c.Settings.OpenCode {
		out["setting."+k] = mustJSON(v)
	}
	return out
}

// desiredTUI maps each [tui.opencode] key to its tui.* state key (tui.json).
func desiredTUI(c *config.Config) map[string]string {
	out := map[string]string{}
	for k, v := range c.TUI.OpenCode {
		out["tui."+k] = mustJSON(v)
	}
	return out
}

// Document-path mappings for each structured-document namespace. Config-supplied
// names are escaped so a name with dots/@/|/# addresses the literal key.
func mcpDocPath(key string) string     { return "mcp." + jsonutil.EscapePath(trim(key, "mcp.")) }
func settingDocPath(key string) string { return jsonutil.EscapePath(trim(key, "setting.")) }
func tuiDocPath(key string) string     { return jsonutil.EscapePath(trim(key, "tui.")) }

func (a *Adapter) Plan(c *config.Config, st *state.State) (adapter.ChangeSet, error) {
	skills, err := c.ExpandedSkillEntriesForTool("opencode")
	if err != nil {
		return adapter.ChangeSet{}, err
	}
	a.skills = skills
	commands, err := c.ExpandedCommandEntriesForTool("opencode")
	if err != nil {
		return adapter.ChangeSet{}, err
	}
	a.commands = commands
	subagents, err := c.ExpandedSubagentEntriesForTool("opencode")
	if err != nil {
		return adapter.ChangeSet{}, err
	}
	a.subagents = subagents
	doc, err := readStandardized(a.cfgFile())
	if err != nil {
		return adapter.ChangeSet{}, err
	}
	tuiDoc, err := readStandardized(a.tuiFile())
	if err != nil {
		return adapter.ChangeSet{}, err
	}
	cs := adapter.ChangeSet{Tool: "opencode"}

	// Structured-document namespaces go through the shared projection contract:
	// mcp./setting.* live in opencode.jsonc; tui.* lives in tui.json. Each Project
	// call prunes only its own recorded keys, so the generic delete loop below no
	// longer touches these prefixes. plugin.* stays bespoke (array membership).
	codec := jsoncodec.Codec{}
	des := a.desiredMCPs(c)
	cs.Changes = append(cs.Changes, structproj.Project("opencode", "mcp.", des, doc, st, codec, mcpDocPath)...)
	cs.Changes = append(cs.Changes, structproj.Project("opencode", "setting.", desiredSettings(c), doc, st, codec, settingDocPath)...)
	cs.Changes = append(cs.Changes, structproj.Project("opencode", "tui.", desiredTUI(c), tuiDoc, st, codec, tuiDocPath)...)
	for _, pl := range c.Plugins.OpenCode {
		src := pl.Source
		_, inState := st.Get("opencode", "plugin."+src)
		if !pl.IsEnabled() {
			// Disabled: ensure absent, but only ever remove a homonto-managed
			// entry (recorded in state). A present-but-unmanaged source, or one
			// already absent, is left untouched — no change emitted.
			if arrayHas(doc, "plugin", src) && inState {
				cs.Changes = append(cs.Changes, adapter.Change{Action: "delete", Key: "plugin." + src, Old: adapter.SecretRedaction})
			}
			continue
		}
		if arrayHas(doc, "plugin", src) {
			// Present on disk. If recorded, steady-state noop; otherwise adopt it
			// into state so pruning and drift can see it (plugin names are plain,
			// never secret-bearing).
			if inState {
				cs.Changes = append(cs.Changes, adapter.Change{Action: "noop", Key: "plugin." + src})
			} else {
				cs.Changes = append(cs.Changes, adapter.Change{Action: "adopt", Key: "plugin." + src, New: mustJSON(src)})
			}
		} else {
			cs.Changes = append(cs.Changes, adapter.Change{Action: "create", Key: "plugin." + src, New: mustJSON(src)})
		}
	}
	// File-projection namespaces go through the shared symlink contract: each
	// Project call emits create/relocate/relink + adopt for its links and plans
	// NO deletes — the generic delete loop below stays the single source of
	// file-prefix deletes.
	roots := a.managedRoots()
	skillChanges, err := fileproj.Project("opencode", a.skillFileLinks(), st, roots)
	if err != nil {
		return adapter.ChangeSet{}, err
	}
	cs.Changes = append(cs.Changes, skillChanges...)
	commandChanges, err := fileproj.Project("opencode", a.commandFileLinks(), st, roots)
	if err != nil {
		return adapter.ChangeSet{}, err
	}
	cs.Changes = append(cs.Changes, commandChanges...)
	subagentChanges, err := fileproj.Project("opencode", a.subagentFileLinks(), st, roots)
	if err != nil {
		return adapter.ChangeSet{}, err
	}
	cs.Changes = append(cs.Changes, subagentChanges...)
	// Orphans: a state key no longer declared in config is de-declared — plan a
	// delete. (A declared key missing from disk is drift, handled above.) Old is
	// always redacted: a removed key's provenance is stale by definition.
	declared := map[string]bool{}
	for k := range des {
		declared[k] = true
	}
	for k := range c.Settings.OpenCode {
		declared["setting."+k] = true
	}
	for k := range c.TUI.OpenCode {
		declared["tui."+k] = true
	}
	for _, pl := range c.Plugins.OpenCode {
		declared["plugin."+pl.Source] = true
	}
	for _, entry := range a.skills {
		declared["skill."+entry.Name] = true
	}
	for _, entry := range a.commands {
		declared["command."+entry.Name] = true
	}
	for _, entry := range a.subagents {
		declared["subagent."+entry.Name] = true
	}
	// Copy-mode subagents are managed content files: surface create/update/prune
	// and abort on a foreign-file conflict; Apply reconciles them in a dedicated
	// pass. subagentcopy.* is outside managedPrefix so the generic delete loop
	// never touches it.
	copyOps, err := a.planCopyOps(st)
	if err != nil {
		return adapter.ChangeSet{}, err
	}
	for _, op := range copyOps {
		name := copyproj.Name(op.Dst)
		switch op.Action {
		case copyfile.Conflict:
			return adapter.ChangeSet{}, fmt.Errorf("opencode: %s exists and is not a homonto-managed copy-mode subagent; not overwriting", op.Dst)
		case copyfile.Create:
			cs.Changes = append(cs.Changes, adapter.Change{Action: "create", Key: "subagentcopy." + name, New: op.Dst})
		case copyfile.Update, copyfile.LocalEdit:
			cs.Changes = append(cs.Changes, adapter.Change{Action: "update", Key: "subagentcopy." + name, New: op.Dst})
		case copyfile.Prune:
			cs.Changes = append(cs.Changes, adapter.Change{Action: "delete", Key: "subagentcopy." + name, Old: op.Dst})
		}
	}
	// The generic prune covers plugin.* (array membership) and the file-projection
	// prefixes; the structured prefixes (mcp./setting./tui.) are pruned by their
	// structproj.Project calls above (avoiding a double delete).
	for _, k := range st.Keys("opencode") {
		if declared[k] || !managedPrefix(k) {
			continue
		}
		cs.Changes = append(cs.Changes, adapter.Change{Action: "delete", Key: k, Old: adapter.SecretRedaction})
	}
	// Keys come from map iteration (random order); a plan must render the
	// same way every run. Keys are unique within a changeset.
	sort.SliceStable(cs.Changes, func(i, j int) bool { return cs.Changes[i].Key < cs.Changes[j].Key })
	return cs, nil
}

// skillFileLinks builds the desired managed skill symlinks for the fileproj
// contract: destination, content source, state key, and the same-named link at
// the other scope (Inactive is "" when there is nothing to relocate from).
func (a *Adapter) skillFileLinks() []fileproj.Link {
	var out []fileproj.Link
	for _, e := range a.skills {
		inact := ""
		if d := a.inactiveSkillsDir(e.Resource.Scope); d != "" {
			inact = filepath.Join(d, e.Name)
		}
		out = append(out, fileproj.Link{
			Dst:      filepath.Join(a.skillsDir(e.Resource.Scope), e.Name),
			Src:      a.skillSource(e),
			Key:      "skill." + e.Name,
			Inactive: inact,
		})
	}
	return out
}

// filterChanges returns the subset of changes whose keys are in prefix, so each
// structproj namespace applies only the changes it owns.
func filterChanges(changes []adapter.Change, prefix string) []adapter.Change {
	var out []adapter.Change
	for _, c := range changes {
		if strings.HasPrefix(c.Key, prefix) {
			out = append(out, c)
		}
	}
	return out
}

// ObserveHashes hashes the current on-disk value of every recorded key still
// present, so an unchanged key reproduces its Entry.Applied (mirroring claude,
// as far as opencode's data model allows). Only hashes escape — raw values
// (possibly resolved secrets) never leave the adapter.
func (a *Adapter) ObserveHashes(st *state.State) (map[string]string, error) {
	doc, err := readStandardized(a.cfgFile())
	if err != nil {
		return nil, err
	}
	tuiDoc, err := readStandardized(a.tuiFile())
	if err != nil {
		return nil, err
	}
	codec := jsoncodec.Codec{}
	out := map[string]string{}
	// Structured-document keys (mcp./setting.* in opencode.jsonc; tui.* in
	// tui.json) re-hash their on-disk value through the shared contract.
	for k, v := range structproj.Observe("opencode", "mcp.", doc, st, codec, mcpDocPath) {
		out[k] = v
	}
	for k, v := range structproj.Observe("opencode", "setting.", doc, st, codec, settingDocPath) {
		out[k] = v
	}
	for k, v := range structproj.Observe("opencode", "tui.", tuiDoc, st, codec, tuiDocPath) {
		out[k] = v
	}
	// File-projection keys (skill./command./subagent.*) live on disk as symlinks;
	// each re-hashes its recorded link through the shared contract, reading at the
	// recorded dst so a pending scope switch is not misread as drift.
	for k, v := range fileproj.Observe("opencode", "skill.", st) {
		out[k] = v
	}
	for k, v := range fileproj.Observe("opencode", "command.", st) {
		out[k] = v
	}
	for k, v := range fileproj.Observe("opencode", "subagent.", st) {
		out[k] = v
	}
	for _, key := range st.Keys("opencode") {
		switch {
		case hasPrefix(key, "plugin."):
			// Plugins are array membership with no scalar to re-hash: presence
			// means unchanged by definition, so return the key's own Applied.
			if arrayHas(doc, "plugin", trim(key, "plugin.")) {
				if e, ok := st.Get("opencode", key); ok {
					out[key] = e.Applied
				}
			}
		case hasPrefix(key, "subagentcopy."):
			// A copy-mode subagent lives on disk as a real file; its Applied is the
			// content hash and Desired holds the dst path.
			e, ok := st.Get("opencode", key)
			if !ok {
				continue
			}
			content, err := os.ReadFile(e.Desired)
			if err != nil {
				continue
			}
			out[key] = copyfile.Hash(content)
		}
		// absent from disk → omit
	}
	return out, nil
}

func (a *Adapter) Apply(cs adapter.ChangeSet, res *secret.Resolver, st *state.State) error {
	doc, err := readStandardized(a.cfgFile())
	if err != nil {
		return err
	}
	tuiDoc, err := readStandardized(a.tuiFile())
	if err != nil {
		return err
	}
	// Write opencode.jsonc only when a managed key in it actually changed.
	// adopt/noop are state-only and must leave the file byte-for-byte untouched
	// (JSONC comments preserved); skill.* is symlink work, not JSON. tuiChanged
	// gates tui.json's write independently, so a change to one file never
	// rewrites the other.
	codec := jsoncodec.Codec{}
	// Structured-document prefixes go through the shared contract. Order matters
	// for byte-identical output: mcp./plugin./setting.* all live in opencode.jsonc,
	// and the prior single sorted-change loop appended them in mcp < plugin <
	// setting order — so apply them in that order too. tui.* lives in tui.json.
	doc, docChanged, err := structproj.Apply("opencode", "mcp.", filterChanges(cs.Changes, "mcp."), doc, codec, res, st, mcpDocPath)
	if err != nil {
		return err
	}
	// plugin.* is bespoke array membership (structproj's keyed codec cannot model
	// it): create/update adds the element, delete removes it, adopt records state.
	for _, c := range filterChanges(cs.Changes, "plugin.") {
		switch c.Action {
		case "noop":
			continue
		case "adopt":
			// Records a pre-existing membership into state without touching disk.
			val, err := res.ResolveJSON(c.New)
			if err != nil {
				return err
			}
			st.Set("opencode", c.Key, c.New, secret.Hash(jsonutil.Canonical(mustJSON(val))))
		case "delete":
			if doc, err = jsonutil.RemoveArrayElem(doc, "plugin", trim(c.Key, "plugin.")); err != nil {
				return err
			}
			docChanged = true
			st.Delete("opencode", c.Key)
		default: // create | update
			val, err := res.ResolveJSON(c.New)
			if err != nil {
				return err
			}
			if doc, err = jsonutil.EnsureArrayElem(doc, "plugin", trim(c.Key, "plugin.")); err != nil {
				return err
			}
			docChanged = true
			st.Set("opencode", c.Key, c.New, secret.Hash(jsonutil.Canonical(mustJSON(val))))
		}
	}
	{
		var ch bool
		doc, ch, err = structproj.Apply("opencode", "setting.", filterChanges(cs.Changes, "setting."), doc, codec, res, st, settingDocPath)
		if err != nil {
			return err
		}
		docChanged = docChanged || ch
	}
	tuiDoc, tuiChanged, err := structproj.Apply("opencode", "tui.", filterChanges(cs.Changes, "tui."), tuiDoc, codec, res, st, tuiDocPath)
	if err != nil {
		return err
	}
	// File-projection keys (skill./command./subagent.): adopt records state only;
	// delete removes the managed symlink. Their create/update are handled by the
	// fileproj.ApplyLinks pass below; noop and subagentcopy.* are handled
	// elsewhere. The fallback recovers a de-declared key's on-disk dst at user
	// scope when state lacks a recorded dst, matching the prior inline behavior.
	roots := a.managedRoots()
	if err := fileproj.ApplyState("opencode", filterChanges(cs.Changes, "skill."), st, roots, func(k string) string {
		return filepath.Join(a.skillsDir("user"), trim(k, "skill."))
	}); err != nil {
		return err
	}
	if err := fileproj.ApplyState("opencode", filterChanges(cs.Changes, "command."), st, roots, func(k string) string {
		return filepath.Join(a.commandsDir("user"), trim(k, "command.")+".md")
	}); err != nil {
		return err
	}
	if err := fileproj.ApplyState("opencode", filterChanges(cs.Changes, "subagent."), st, roots, func(k string) string {
		return filepath.Join(a.subagentsDir("user"), trim(k, "subagent.")+".md")
	}); err != nil {
		return err
	}
	// Fail fast on link conflicts before writing any file. A conflict in any of
	// the three namespaces must be detected here, before any JSON write or state
	// mutation below — otherwise a command conflict could let Apply partially
	// write opencode.jsonc and commit skill-link state before erroring.
	if err := fileproj.Conflicts(a.skillFileLinks(), roots); err != nil {
		return err
	}
	if err := fileproj.Conflicts(a.commandFileLinks(), roots); err != nil {
		return err
	}
	if err := fileproj.Conflicts(a.subagentFileLinks(), roots); err != nil {
		return err
	}
	// Fail fast on a copy-mode subagent conflict too, before any file is written.
	copyOps, err := a.planCopyOps(st)
	if err != nil {
		return err
	}
	for _, op := range copyOps {
		if op.Action == copyfile.Conflict {
			return fmt.Errorf("opencode: %s exists and is not a homonto-managed copy-mode subagent; not overwriting", op.Dst)
		}
	}
	if docChanged {
		if err := fsutil.WriteAtomic(a.cfgFile(), doc); err != nil {
			return err
		}
	}
	if tuiChanged {
		if err := fsutil.WriteAtomic(a.tuiFile(), tuiDoc); err != nil {
			return err
		}
	}
	// Prune each namespace's inactive-scope orphan (left after a per-resource
	// scope switch), then create the link and record state. Runs after the JSON
	// writes. Only our own managed symlink is ever removed (IsManaged guards it);
	// a foreign file or an absent path is left untouched.
	if err := fileproj.ApplyLinks("opencode", a.skillFileLinks(), st, roots); err != nil {
		return err
	}
	if err := fileproj.ApplyLinks("opencode", a.commandFileLinks(), st, roots); err != nil {
		return err
	}
	if err := fileproj.ApplyLinks("opencode", a.subagentFileLinks(), st, roots); err != nil {
		return err
	}
	// Reconcile copy-mode subagent content files (write/update/prune + state).
	if err := a.applyCopySubagents(st); err != nil {
		return err
	}
	return nil
}
