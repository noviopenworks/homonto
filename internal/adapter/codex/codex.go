// Package codex is the pilot third-party adapter built on the adapter contract
// (internal/adapter/structproj) and the TOML codec (internal/tomlutil). It
// projects declared MCP servers targeting "codex" into ~/.codex/config.toml as
// [mcp_servers.<name>] tables, surgically and idempotently, without duplicating
// the Claude/OpenCode control flow — the adapter supplies only its file path,
// key mapping, and codec; all plan/apply/observe logic lives in the contract.
package codex

import (
	"os"
	"path/filepath"
	"slices"

	"github.com/noviopenworks/homonto/internal/adapter"
	"github.com/noviopenworks/homonto/internal/adapter/structproj"
	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/fsutil"
	"github.com/noviopenworks/homonto/internal/secret"
	"github.com/noviopenworks/homonto/internal/state"
	"github.com/noviopenworks/homonto/internal/tomlutil"
)

// Tool is the adapter/target name.
const Tool = "codex"

// keyPrefix namespaces MCP state keys; docPath maps them to config.toml.
const keyPrefix = "mcp."

// Adapter projects config into Codex's ~/.codex/config.toml.
type Adapter struct {
	home string
}

// New builds a Codex adapter rooted at home ($HOME).
func New(home string) *Adapter { return &Adapter{home: home} }

// Name identifies the tool.
func (a *Adapter) Name() string { return Tool }

func (a *Adapter) configTOML() string { return filepath.Join(a.home, ".codex", "config.toml") }

// docPath maps a state key (mcp.<name>) to the config.toml table path,
// quoting the server name so a name containing dots is one table key, not nested
// tables (mirrors the built-in adapters' EscapePath behavior).
func docPath(stateKey string) string {
	return "mcp_servers." + tomlutil.QuoteSegment(stateKey[len(keyPrefix):])
}

// codec adapts the tomlutil package functions to structproj.Codec.
type codec struct{}

func (codec) EnsureRoot(doc []byte) ([]byte, error)       { return tomlutil.EnsureRoot(doc) }
func (codec) Get(doc []byte, p string) (string, bool)     { return tomlutil.Get(doc, p) }
func (codec) Set(doc []byte, p, v string) ([]byte, error) { return tomlutil.Set(doc, p, v) }
func (codec) Delete(doc []byte, p string) ([]byte, error) { return tomlutil.Delete(doc, p) }
func (codec) Canonical(v string) string                   { return tomlutil.Canonical(v) }

// desired maps each MCP that explicitly targets codex to its mcp_servers.<name>
// value. Codex is opt-in: an MCP with no explicit targets defaults to
// claude+opencode and is skipped here.
func (a *Adapter) desired(c *config.Config) map[string]string {
	out := map[string]string{}
	for name, m := range c.MCPs {
		if !targetsCodex(m.Targets) || len(m.Command) == 0 {
			continue
		}
		obj := map[string]any{"command": m.Command[0]}
		if len(m.Command) > 1 {
			obj["args"] = m.Command[1:]
		}
		if len(m.Env) > 0 {
			obj["env"] = m.Env
		}
		out[keyPrefix+name] = structproj.MustJSON(obj)
	}
	return out
}

func targetsCodex(targets []string) bool {
	return slices.Contains(targets, Tool)
}

// Plan diffs desired MCP servers against config.toml and recorded state.
func (a *Adapter) Plan(c *config.Config, st *state.State) (adapter.ChangeSet, error) {
	disk, err := a.read()
	if err != nil {
		return adapter.ChangeSet{}, err
	}
	changes := structproj.Project(Tool, keyPrefix, a.desired(c), disk, st, codec{}, docPath)
	return adapter.ChangeSet{Tool: Tool, Changes: changes}, nil
}

// Apply writes managed MCP tables into config.toml (only when a managed key
// changed), preserving unmanaged content, and records state.
func (a *Adapter) Apply(cs adapter.ChangeSet, res *secret.Resolver, st *state.State) error {
	disk, err := a.read()
	if err != nil {
		return err
	}
	doc, changed, err := structproj.Apply(Tool, keyPrefix, cs.Changes, disk, codec{}, res, st, docPath)
	if err != nil {
		return err
	}
	if !changed {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(a.configTOML()), 0o755); err != nil {
		return err
	}
	return fsutil.WriteAtomic(a.configTOML(), doc)
}

// ObserveHashes re-hashes each recorded MCP key still present in config.toml.
func (a *Adapter) ObserveHashes(st *state.State) (map[string]string, error) {
	disk, err := a.read()
	if err != nil {
		return nil, err
	}
	return structproj.Observe(Tool, keyPrefix, disk, st, codec{}, docPath), nil
}

// read returns the raw config.toml, or empty bytes when absent.
func (a *Adapter) read() ([]byte, error) {
	b, err := os.ReadFile(a.configTOML())
	if os.IsNotExist(err) {
		return []byte{}, nil
	}
	return b, err
}
