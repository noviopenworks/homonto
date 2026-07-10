package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/noviopenworks/homonto/internal/adapter"
	"github.com/noviopenworks/homonto/internal/commandpath"
	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/fsutil"
	"github.com/noviopenworks/homonto/internal/jsonutil"
	"github.com/noviopenworks/homonto/internal/link"
	"github.com/noviopenworks/homonto/internal/secret"
	"github.com/noviopenworks/homonto/internal/skillpath"
	"github.com/noviopenworks/homonto/internal/state"
	"github.com/noviopenworks/homonto/internal/subagentpath"
)

// Adapter projects desired config into Claude Code's files under home.
type Adapter struct {
	home                string
	content             string
	catalogRoot         string // materialized builtin catalog root (.homonto/catalog/skills)
	commandCatalogRoot  string // materialized builtin command root (.homonto/catalog/commands)
	subagentCatalogRoot string // materialized builtin subagent root (.homonto/catalog/subagents)
	projectRoot         string // directory of homonto.toml; used for project-scope resources
	skills              []config.NamedResource
	commands            []config.NamedResource
	subagents           []config.NamedResource
}

// New builds a Claude adapter at user scope. home is the $HOME root; content
// holds owned skills. Use WithProjectRoot to install project-scope skills.
func New(home, content string) *Adapter { return &Adapter{home: home, content: content} }

// WithProjectRoot sets the project root (the homonto.toml directory). It is
// used for project-scope resource placement. MCP servers and settings always
// project under home.
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

func (a *Adapter) Name() string { return "claude" }

func (a *Adapter) claudeJSON() string   { return filepath.Join(a.home, ".claude.json") }
func (a *Adapter) settingsJSON() string { return filepath.Join(a.home, ".claude", "settings.json") }

// skillsDir is the directory owned-skill symlinks live in for the given scope.
func (a *Adapter) skillsDir(scope string) string {
	return skillpath.Dir("claude", scope, a.home, a.projectRoot)
}

// inactiveSkillsDir is the other scope's skills directory — where a link may
// linger after a per-resource scope switch. It returns "" when there is nothing
// meaningful to relocate from: no project root is known, or the two scopes
// resolve to the same directory (a homonto.toml that sits in $HOME).
func (a *Adapter) inactiveSkillsDir(scope string) string {
	if a.projectRoot == "" {
		return ""
	}
	d := skillpath.Dir("claude", skillpath.Other(scope), a.home, a.projectRoot)
	if d == a.skillsDir(scope) {
		return ""
	}
	return d
}

// links maps each owned skill's destination to its content source. Each skill
// resource carries its own scope, so dst is computed per entry.
func (a *Adapter) links() map[string]string {
	out := map[string]string{}
	for _, entry := range a.skills {
		out[filepath.Join(a.skillsDir(entry.Resource.Scope), entry.Name)] = a.skillSource(entry)
	}
	return out
}

// commandsDir is the directory owned-command symlinks live in for the scope.
func (a *Adapter) commandsDir(scope string) string {
	return commandpath.Dir("claude", scope, a.home, a.projectRoot)
}

// inactiveCommandsDir is the other scope's commands directory — where a link
// may linger after a per-resource scope switch. It returns "" when nothing
// meaningful can be relocated (no project root, or both scopes resolve equal).
func (a *Adapter) inactiveCommandsDir(scope string) string {
	if a.projectRoot == "" {
		return ""
	}
	d := commandpath.Dir("claude", skillpath.Other(scope), a.home, a.projectRoot)
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

// commandLinks maps each owned command's destination (<name>.md) to its source.
func (a *Adapter) commandLinks() map[string]string {
	out := map[string]string{}
	for _, entry := range a.commands {
		out[filepath.Join(a.commandsDir(entry.Resource.Scope), entry.Name+".md")] = a.commandSource(entry)
	}
	return out
}

// subagentsDir is the directory owned-subagent symlinks live in for the scope.
func (a *Adapter) subagentsDir(scope string) string {
	return subagentpath.Dir("claude", scope, a.home, a.projectRoot)
}

// inactiveSubagentsDir is the other scope's subagent directory — where a link
// may linger after a per-resource scope switch. It returns "" when nothing
// meaningful can be relocated (no project root, or both scopes resolve equal).
func (a *Adapter) inactiveSubagentsDir(scope string) string {
	if a.projectRoot == "" {
		return ""
	}
	d := subagentpath.Dir("claude", skillpath.Other(scope), a.home, a.projectRoot)
	if d == a.subagentsDir(scope) {
		return ""
	}
	return d
}

// subagentSource resolves a subagent entry's on-disk file by source scheme:
// builtin:<n> from the materialized subagent root (<n>.md), otherwise the local
// content dir (homonto/subagents/<n>.md).
func (a *Adapter) subagentSource(entry config.NamedResource) string {
	if s := entry.Resource.Source; strings.HasPrefix(s, "builtin:") {
		return filepath.Join(a.subagentCatalogRoot, strings.TrimPrefix(s, "builtin:")+".md")
	}
	return filepath.Join(a.content, "subagents", localSourceName(entry.Resource.Source, entry.Name)+".md")
}

// subagentLinks maps each owned subagent's destination (<name>.md) to its source.
func (a *Adapter) subagentLinks() map[string]string {
	out := map[string]string{}
	for _, entry := range a.subagents {
		out[filepath.Join(a.subagentsDir(entry.Resource.Scope), entry.Name+".md")] = a.subagentSource(entry)
	}
	return out
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

// desired returns managed key -> unresolved JSON-encoded desired value.
func (a *Adapter) desired(c *config.Config) map[string]string {
	out := map[string]string{}
	for name, m := range c.MCPs {
		if !contains(m.TargetsOrAll(), "claude") {
			continue
		}
		if len(m.Command) == 0 {
			continue
		}
		// Claude Code's real schema: command is a string with a separate args
		// array (matching `claude mcp add` output; empty keys omitted).
		obj := map[string]any{"type": "stdio", "command": m.Command[0]}
		if len(m.Command) > 1 {
			obj["args"] = m.Command[1:]
		}
		if len(m.Env) > 0 {
			obj["env"] = m.Env
		}
		out["mcp."+name] = mustJSON(obj)
	}
	for k, v := range c.Settings.Claude {
		out["setting."+k] = mustJSON(v)
	}
	for _, p := range c.Plugins.Claude {
		out["plugin."+p] = `true`
	}
	return out
}

func (a *Adapter) Plan(c *config.Config, st *state.State) (adapter.ChangeSet, error) {
	skills, err := c.ExpandedSkillEntriesForTool("claude")
	if err != nil {
		return adapter.ChangeSet{}, err
	}
	a.skills = skills
	commands, err := c.ExpandedCommandEntriesForTool("claude")
	if err != nil {
		return adapter.ChangeSet{}, err
	}
	a.commands = commands
	subagents, err := c.ExpandedSubagentEntriesForTool("claude")
	if err != nil {
		return adapter.ChangeSet{}, err
	}
	a.subagents = subagents
	cur, err := a.current()
	if err != nil {
		return adapter.ChangeSet{}, err
	}
	cs := adapter.ChangeSet{Tool: "claude"}
	des := a.desired(c)
	for key, want := range des {
		disk, hasDisk := cur[key]
		e, inState := st.Get("claude", key)
		switch {
		case !hasDisk:
			cs.Changes = append(cs.Changes, adapter.Change{Action: "create", Key: key, New: want})
		case !secret.ContainsRef(want):
			if jsonutil.Canonical(disk) == jsonutil.Canonical(want) {
				// Disk already matches desired. A true noop requires state to also
				// already record this exact on-disk value; otherwise adopt it so
				// pruning and drift can see it and the stale/absent Applied hash is
				// refreshed (mirrors the secret branch below; secret keys never
				// reach this branch).
				if inState && e.Applied == secret.Hash(jsonutil.Canonical(disk)) {
					cs.Changes = append(cs.Changes, adapter.Change{Action: "noop", Key: key})
				} else {
					cs.Changes = append(cs.Changes, adapter.Change{Action: "adopt", Key: key, New: want})
				}
			} else {
				old := disk
				// Never print the on-disk value when it may be a resolved secret:
				// either the key was previously a secret, or it is not in state at
				// all (unknown provenance — a lost state.json must not cause leaks).
				if !inState || secret.ContainsRef(e.Desired) {
					old = adapter.SecretRedaction
				}
				cs.Changes = append(cs.Changes, adapter.Change{Action: "update", Key: key, Old: old, New: want})
			}
		default: // secret-bearing key: never expose the on-disk resolved value
			if inState && e.Desired == want && e.Applied == secret.Hash(jsonutil.Canonical(disk)) {
				cs.Changes = append(cs.Changes, adapter.Change{Action: "noop", Key: key})
			} else {
				cs.Changes = append(cs.Changes, adapter.Change{Action: "update", Key: key, Old: adapter.SecretRedaction, New: want})
			}
		}
	}
	ops, err := link.Plan(a.links(), a.managedRoots()...)
	if err != nil {
		return adapter.ChangeSet{}, err
	}
	entryByName := map[string]config.NamedResource{}
	for _, entry := range a.skills {
		entryByName[entry.Name] = entry
	}
	for _, op := range ops {
		name := filepath.Base(op.Dst)
		entry := entryByName[name]
		inactive := a.inactiveSkillsDir(entry.Resource.Scope)
		// A create whose same-named link still exists (as our managed symlink) at
		// the other scope is a scope switch: render it as a relocate so the move —
		// and the prune of the old link Apply performs — is visible before confirm.
		if op.Cur == "" && inactive != "" && link.IsManaged(filepath.Join(inactive, name), a.managedRoots()...) {
			cs.Changes = append(cs.Changes, adapter.Change{Action: "update", Key: "skill." + name, Old: filepath.Join(inactive, name), New: op.Dst + " -> " + op.Src})
		} else if op.Cur == "" {
			cs.Changes = append(cs.Changes, adapter.Change{Action: "create", Key: "skill." + name, New: op.Dst + " -> " + op.Src})
		} else {
			cs.Changes = append(cs.Changes, adapter.Change{Action: "update", Key: "skill." + name, Old: op.Cur, New: op.Src})
		}
	}
	// Adopt a correct-but-unrecorded skill link — one already on disk pointing at
	// its content, but absent from state (or stale). link.Plan omits a correct
	// link, so without this a lost state.json for a skills-only config could never
	// be rebuilt (apply short-circuits with no change). Mirrors mcp/setting/plugin
	// adoption: state-only, the on-disk link is left untouched.
	opDst := map[string]bool{}
	for _, op := range ops {
		opDst[op.Dst] = true
	}
	for _, entry := range a.skills {
		name := entry.Name
		dst := filepath.Join(a.skillsDir(entry.Resource.Scope), name)
		if opDst[dst] {
			continue // a create/relink/relocate already covers it
		}
		src := a.skillSource(entry)
		if tgt, err := os.Readlink(dst); err != nil || tgt != src {
			continue // not a correct link into content
		}
		if e, ok := st.Get("claude", "skill."+name); ok && e.Applied == secret.Hash(dst+" -> "+src) {
			continue // already recorded → a true noop, nothing to do
		}
		cs.Changes = append(cs.Changes, adapter.Change{Action: "adopt", Key: "skill." + name, New: dst + " -> " + src})
	}
	// ---- command links (parallel to skills) ----
	cmdOps, err := link.Plan(a.commandLinks(), a.managedRoots()...)
	if err != nil {
		return adapter.ChangeSet{}, err
	}
	cmdByName := map[string]config.NamedResource{}
	for _, entry := range a.commands {
		cmdByName[entry.Name] = entry
	}
	for _, op := range cmdOps {
		name := strings.TrimSuffix(filepath.Base(op.Dst), ".md")
		entry := cmdByName[name]
		inactive := a.inactiveCommandsDir(entry.Resource.Scope)
		if op.Cur == "" && inactive != "" && link.IsManaged(filepath.Join(inactive, name+".md"), a.managedRoots()...) {
			cs.Changes = append(cs.Changes, adapter.Change{Action: "update", Key: "command." + name, Old: filepath.Join(inactive, name+".md"), New: op.Dst + " -> " + op.Src})
		} else if op.Cur == "" {
			cs.Changes = append(cs.Changes, adapter.Change{Action: "create", Key: "command." + name, New: op.Dst + " -> " + op.Src})
		} else {
			cs.Changes = append(cs.Changes, adapter.Change{Action: "update", Key: "command." + name, Old: op.Cur, New: op.Src})
		}
	}
	cmdOpDst := map[string]bool{}
	for _, op := range cmdOps {
		cmdOpDst[op.Dst] = true
	}
	for _, entry := range a.commands {
		dst := filepath.Join(a.commandsDir(entry.Resource.Scope), entry.Name+".md")
		if cmdOpDst[dst] {
			continue
		}
		src := a.commandSource(entry)
		if tgt, err := os.Readlink(dst); err != nil || tgt != src {
			continue
		}
		if e, ok := st.Get("claude", "command."+entry.Name); ok && e.Applied == secret.Hash(dst+" -> "+src) {
			continue
		}
		cs.Changes = append(cs.Changes, adapter.Change{Action: "adopt", Key: "command." + entry.Name, New: dst + " -> " + src})
	}
	// ---- subagent links (parallel to commands) ----
	subOps, err := link.Plan(a.subagentLinks(), a.managedRoots()...)
	if err != nil {
		return adapter.ChangeSet{}, err
	}
	subByName := map[string]config.NamedResource{}
	for _, entry := range a.subagents {
		subByName[entry.Name] = entry
	}
	for _, op := range subOps {
		name := strings.TrimSuffix(filepath.Base(op.Dst), ".md")
		entry := subByName[name]
		inactive := a.inactiveSubagentsDir(entry.Resource.Scope)
		if op.Cur == "" && inactive != "" && link.IsManaged(filepath.Join(inactive, name+".md"), a.managedRoots()...) {
			cs.Changes = append(cs.Changes, adapter.Change{Action: "update", Key: "subagent." + name, Old: filepath.Join(inactive, name+".md"), New: op.Dst + " -> " + op.Src})
		} else if op.Cur == "" {
			cs.Changes = append(cs.Changes, adapter.Change{Action: "create", Key: "subagent." + name, New: op.Dst + " -> " + op.Src})
		} else {
			cs.Changes = append(cs.Changes, adapter.Change{Action: "update", Key: "subagent." + name, Old: op.Cur, New: op.Src})
		}
	}
	subOpDst := map[string]bool{}
	for _, op := range subOps {
		subOpDst[op.Dst] = true
	}
	for _, entry := range a.subagents {
		dst := filepath.Join(a.subagentsDir(entry.Resource.Scope), entry.Name+".md")
		if subOpDst[dst] {
			continue
		}
		src := a.subagentSource(entry)
		if tgt, err := os.Readlink(dst); err != nil || tgt != src {
			continue
		}
		if e, ok := st.Get("claude", "subagent."+entry.Name); ok && e.Applied == secret.Hash(dst+" -> "+src) {
			continue
		}
		cs.Changes = append(cs.Changes, adapter.Change{Action: "adopt", Key: "subagent." + entry.Name, New: dst + " -> " + src})
	}
	// Orphans: a state key no longer declared in config is de-declared — plan a
	// delete. (A declared key missing from disk is drift, handled above.) Old is
	// always redacted: a removed key's provenance is stale by definition.
	declared := map[string]bool{}
	for k := range des {
		declared[k] = true
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
	for _, k := range st.Keys("claude") {
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

// current reads existing managed values from disk, keyed like desired().
func (a *Adapter) current() (map[string]string, error) {
	out := map[string]string{}
	mj, err := readStandardized(a.claudeJSON())
	if err != nil {
		return nil, err
	}
	sj, err := readStandardized(a.settingsJSON())
	if err != nil {
		return nil, err
	}
	for k, v := range objMembers(mj, "mcpServers") {
		out["mcp."+k] = v
	}
	for k, v := range objMembers(sj, "enabledPlugins") {
		out["plugin."+k] = v
	}
	var m map[string]json.RawMessage
	_ = json.Unmarshal(sj, &m)
	for k, raw := range m {
		if k == "mcpServers" || k == "enabledPlugins" {
			continue
		}
		out["setting."+k] = string(raw)
	}
	return out, nil
}

// ObserveHashes hashes the current on-disk value of every recorded key still
// present, so an unchanged key reproduces its Entry.Applied (see the plan's
// noop identity: Applied == secret.Hash(jsonutil.Canonical(disk))). Only hashes
// escape — raw values (possibly resolved secrets) never leave the adapter.
func (a *Adapter) ObserveHashes(st *state.State) (map[string]string, error) {
	cur, err := a.current()
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, key := range st.Keys("claude") {
		if hasPrefix(key, "skill.") {
			// skill.* lives on disk as a symlink, not a JSON value. Its Applied was
			// stored as Hash(dst + " -> " + src); reproduce it by reading the link at
			// the dst state recorded — NOT at the current scope's skillsDir. A pending
			// [skills] scope switch changes skillsDir but leaves the applied link in
			// place; reading the new scope's (empty) location would make an intact old
			// link look "missing" (false drift) instead of a pending relocation Plan
			// already surfaces.
			e, ok := st.Get("claude", key)
			if !ok {
				continue
			}
			dst, ok := recordedDst(e.Desired)
			if !ok {
				continue
			}
			target, err := os.Readlink(dst)
			if err != nil {
				continue // missing or not a symlink → omit (engine infers missing)
			}
			out[key] = secret.Hash(dst + " -> " + target)
			continue
		}
		if hasPrefix(key, "command.") {
			e, ok := st.Get("claude", key)
			if !ok {
				continue
			}
			dst, ok := recordedDst(e.Desired)
			if !ok {
				continue
			}
			target, err := os.Readlink(dst)
			if err != nil {
				continue
			}
			out[key] = secret.Hash(dst + " -> " + target)
			continue
		}
		if hasPrefix(key, "subagent.") {
			e, ok := st.Get("claude", key)
			if !ok {
				continue
			}
			dst, ok := recordedDst(e.Desired)
			if !ok {
				continue
			}
			target, err := os.Readlink(dst)
			if err != nil {
				continue
			}
			out[key] = secret.Hash(dst + " -> " + target)
			continue
		}
		// mcp.*, setting.*, plugin.* all live in current() as JSON values.
		if v, ok := cur[key]; ok {
			out[key] = secret.Hash(jsonutil.Canonical(v))
		}
		// absent from disk → omit
	}
	return out, nil
}

func (a *Adapter) Apply(cs adapter.ChangeSet, res *secret.Resolver, st *state.State) error {
	mj, err := readStandardized(a.claudeJSON())
	if err != nil {
		return err
	}
	sj, err := readStandardized(a.settingsJSON())
	if err != nil {
		return err
	}
	// Write a tool file only when a managed key living in it actually changed.
	// adopt/noop are state-only and must leave the file byte-for-byte untouched
	// (comments/formatting preserved); skill.* is symlink work, not JSON.
	mjChanged, sjChanged := false, false
	for _, c := range cs.Changes {
		if c.Action == "noop" {
			continue
		}
		if c.Action == "adopt" {
			// A skill adoption records a correct-but-unrecorded symlink into state
			// without touching disk; its value is "dst -> src", not JSON, so it is
			// recorded exactly like a freshly linked skill (Hash of "dst -> src").
			if hasPrefix(c.Key, "skill.") {
				st.Set("claude", c.Key, c.New, secret.Hash(c.New))
				continue
			}
			if hasPrefix(c.Key, "command.") {
				st.Set("claude", c.Key, c.New, secret.Hash(c.New))
				continue
			}
			if hasPrefix(c.Key, "subagent.") {
				st.Set("claude", c.Key, c.New, secret.Hash(c.New))
				continue
			}
			// Adoption records a pre-existing matching key into state without
			// touching the tool file. The on-disk value already equals want, so
			// the recorded Applied hash equals the hash of the on-disk value.
			val, err := res.ResolveJSON(c.New)
			if err != nil {
				return err
			}
			st.Set("claude", c.Key, c.New, secret.Hash(jsonutil.Canonical(mustJSON(val))))
			continue
		}
		if c.Action == "delete" {
			switch {
			case hasPrefix(c.Key, "mcp."):
				mj, err = jsonutil.DeleteJSON(mj, "mcpServers."+jsonutil.EscapePath(trim(c.Key, "mcp.")))
				mjChanged = true
			case hasPrefix(c.Key, "setting."):
				sj, err = jsonutil.DeleteJSON(sj, jsonutil.EscapePath(trim(c.Key, "setting.")))
				sjChanged = true
			case hasPrefix(c.Key, "plugin."):
				sj, err = jsonutil.DeleteJSON(sj, "enabledPlugins."+jsonutil.EscapePath(trim(c.Key, "plugin.")))
				sjChanged = true
			case hasPrefix(c.Key, "skill."):
				// Only a symlink into our content dir is removed; anything else
				// is a conflict error inside link.Remove. A de-declared skill is
				// no longer in a.skills, so recover the on-disk location from the
				// dst state recorded for it (per-resource scope: each skill lives
				// at exactly one place). Fall back to user scope when state is
				// missing the recorded dst.
				name := trim(c.Key, "skill.")
				dst := ""
				if e, ok := st.Get("claude", c.Key); ok {
					dst, _ = recordedDst(e.Desired)
				}
				if dst == "" {
					dst = filepath.Join(a.skillsDir("user"), name)
				}
				err = link.Remove(dst, a.managedRoots()...)
			case hasPrefix(c.Key, "command."):
				name := trim(c.Key, "command.")
				dst := ""
				if e, ok := st.Get("claude", c.Key); ok {
					dst, _ = recordedDst(e.Desired)
				}
				if dst == "" {
					dst = filepath.Join(a.commandsDir("user"), name+".md")
				}
				err = link.Remove(dst, a.managedRoots()...)
			case hasPrefix(c.Key, "subagent."):
				name := trim(c.Key, "subagent.")
				dst := ""
				if e, ok := st.Get("claude", c.Key); ok {
					dst, _ = recordedDst(e.Desired)
				}
				if dst == "" {
					dst = filepath.Join(a.subagentsDir("user"), name+".md")
				}
				err = link.Remove(dst, a.managedRoots()...)
			}
			if err != nil {
				return err
			}
			st.Delete("claude", c.Key)
			continue
		}
		// skill.* changes are symlink work, handled below — not JSON keys.
		if hasPrefix(c.Key, "skill.") {
			continue
		}
		if hasPrefix(c.Key, "command.") {
			continue
		}
		if hasPrefix(c.Key, "subagent.") {
			continue
		}
		val, err := res.ResolveJSON(c.New)
		if err != nil {
			return err
		}
		// Config-supplied names are escaped so sjson writes the literal key
		// (matching current()'s literal reads) instead of nesting on dots or
		// silently dropping the write on @, | or #.
		switch {
		case hasPrefix(c.Key, "mcp."):
			mj, err = jsonutil.SetJSON(mj, "mcpServers."+jsonutil.EscapePath(trim(c.Key, "mcp.")), val)
			mjChanged = true
		case hasPrefix(c.Key, "setting."):
			sj, err = jsonutil.SetJSON(sj, jsonutil.EscapePath(trim(c.Key, "setting.")), val)
			sjChanged = true
		case hasPrefix(c.Key, "plugin."):
			sj, err = jsonutil.SetJSON(sj, "enabledPlugins."+jsonutil.EscapePath(trim(c.Key, "plugin.")), val)
			sjChanged = true
		}
		if err != nil {
			return err
		}
		// Store the unresolved form + a non-secret hash of the resolved value.
		st.Set("claude", c.Key, c.New, secret.Hash(jsonutil.Canonical(mustJSON(val))))
	}
	// Fail fast on link conflicts before writing any file. Both skill and
	// command conflicts must be detected here, before any JSON write or state
	// mutation below — otherwise a command conflict could let Apply partially
	// write JSON and commit skill-link state before erroring.
	links := a.links()
	if _, err := link.Plan(links, a.managedRoots()...); err != nil {
		return err
	}
	cmdLinks := a.commandLinks()
	if _, err := link.Plan(cmdLinks, a.managedRoots()...); err != nil {
		return err
	}
	subLinks := a.subagentLinks()
	if _, err := link.Plan(subLinks, a.managedRoots()...); err != nil {
		return err
	}
	if mjChanged {
		if err := fsutil.WriteAtomic(a.claudeJSON(), mj); err != nil {
			return err
		}
	}
	if sjChanged {
		if err := fsutil.WriteAtomic(a.settingsJSON(), sj); err != nil {
			return err
		}
	}
	// Prune a link left at a skill's inactive scope after a per-resource scope
	// switch, so no orphan remains. Only our own managed symlink is removed
	// (IsManaged guards it); a foreign file or an absent path is left untouched
	// — never an error. Each skill carries its own scope, so the inactive dir is
	// computed per entry.
	for _, entry := range a.skills {
		inactive := a.inactiveSkillsDir(entry.Resource.Scope)
		if inactive == "" {
			continue
		}
		old := filepath.Join(inactive, entry.Name)
		if link.IsManaged(old, a.managedRoots()...) {
			if err := link.Remove(old, a.managedRoots()...); err != nil {
				return err
			}
		}
	}
	for dst, src := range links {
		if _, err := link.Link(src, dst, a.managedRoots()...); err != nil {
			return err
		}
		// Record the link in state so pruning sees de-declared skills later.
		st.Set("claude", "skill."+filepath.Base(dst), dst+" -> "+src, secret.Hash(dst+" -> "+src))
	}
	// Prune a command link left at its inactive scope after a scope switch.
	for _, entry := range a.commands {
		inactive := a.inactiveCommandsDir(entry.Resource.Scope)
		if inactive == "" {
			continue
		}
		old := filepath.Join(inactive, entry.Name+".md")
		if link.IsManaged(old, a.managedRoots()...) {
			if err := link.Remove(old, a.managedRoots()...); err != nil {
				return err
			}
		}
	}
	for dst, src := range cmdLinks {
		if _, err := link.Link(src, dst, a.managedRoots()...); err != nil {
			return err
		}
		st.Set("claude", "command."+strings.TrimSuffix(filepath.Base(dst), ".md"), dst+" -> "+src, secret.Hash(dst+" -> "+src))
	}
	// Prune a subagent link left at its inactive scope after a scope switch.
	for _, entry := range a.subagents {
		inactive := a.inactiveSubagentsDir(entry.Resource.Scope)
		if inactive == "" {
			continue
		}
		old := filepath.Join(inactive, entry.Name+".md")
		if link.IsManaged(old, a.managedRoots()...) {
			if err := link.Remove(old, a.managedRoots()...); err != nil {
				return err
			}
		}
	}
	for dst, src := range subLinks {
		if _, err := link.Link(src, dst, a.managedRoots()...); err != nil {
			return err
		}
		st.Set("claude", "subagent."+strings.TrimSuffix(filepath.Base(dst), ".md"), dst+" -> "+src, secret.Hash(dst+" -> "+src))
	}
	return nil
}

func readStandardized(path string) ([]byte, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return jsonutil.Standardize(nil)
	}
	if err != nil {
		return nil, err
	}
	doc, err := jsonutil.Standardize(b)
	if err != nil {
		return nil, err
	}
	if err := jsonutil.ObjectRoot(doc); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return doc, nil
}
