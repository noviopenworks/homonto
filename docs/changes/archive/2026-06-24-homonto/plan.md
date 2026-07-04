# homonto Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `homonto`, a personal Go CLI that is the single declarative source of truth for AI coding-tool config, projecting MCPs/skills/plugins/settings into Claude Code and OpenCode via a plan/confirm/apply pipeline.

**Architecture:** Parse `homonto.toml` into one tool-agnostic `Config` (desired state). Each tool is an `Adapter` (`Read`/`Plan`/`Apply`). Shared services: secret resolver (`pass`/env), surgical JSON(C) merge, content linker (symlinks), state store (drift), and a planner/printer. The `apply` command runs a six-stage pipeline (parse → read → plan → confirm → resolve secrets → write).

**Tech Stack:** Go 1.22+, `github.com/spf13/cobra` (CLI), `github.com/pelletier/go-toml/v2` (TOML), `github.com/tidwall/sjson` + `github.com/tidwall/gjson` (surgical JSON edits), `github.com/tailscale/hujson` (JSONC normalize). Standard `testing` for tests.

## Global Constraints

- Module path: `github.com/noviopenworks/homonto`.
- Go version floor: `1.22`.
- Secrets are **referenced, never stored**: config holds `${pass:path}` or `${ENV}` tokens; resolved only at apply time, after confirmation, all-at-once before any write.
- **Surgical merge only**: adapters write only keys they manage and preserve all unmanaged keys in a tool's file.
- **Atomic writes**: every file write goes temp-file → `os.Rename`; `state.json` is written last.
- **Plan output and logs must never contain a resolved secret value.**
- TDD: write the failing test first for every unit. DRY, YAGNI. Commit after each task.
- Owned content is linked via **symlinks**, not copied (v1).

---

### Task 1: Project scaffold + version command

**Files:**
- Create: `go.mod`
- Create: `main.go`
- Create: `internal/cli/root.go`
- Test: `internal/cli/root_test.go`

**Interfaces:**
- Produces: `cli.NewRootCmd() *cobra.Command` — root command, used by `main.go` and every later CLI task. Has persistent flag `--config` (default `homonto.toml`) and a `version` subcommand printing `cli.Version`.

- [ ] **Step 1: Initialize the module and add deps**

Run:
```bash
cd /home/mg/homonto
go mod init github.com/noviopenworks/homonto
go get github.com/spf13/cobra@latest
```
Expected: `go.mod` created with `module github.com/noviopenworks/homonto`, `go 1.22`, and cobra in `require`.

- [ ] **Step 2: Write the failing test**

Create `internal/cli/root_test.go`:
```go
package cli

import (
	"bytes"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"version"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if got := out.String(); got != "homonto "+Version+"\n" {
		t.Fatalf("got %q", got)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/cli/`
Expected: FAIL — `undefined: NewRootCmd` / `undefined: Version`.

- [ ] **Step 4: Write minimal implementation**

Create `internal/cli/root.go`:
```go
package cli

import "github.com/spf13/cobra"

// Version is the homonto build version.
const Version = "0.1.0-dev"

// NewRootCmd builds the root cobra command and registers subcommands.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "homonto",
		Short:         "Declarative config for AI coding tools",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().String("config", "homonto.toml", "path to homonto config")

	root.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print the homonto version",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Printf("homonto %s\n", Version)
			return nil
		},
	})
	return root
}
```

Create `main.go`:
```go
package main

import (
	"fmt"
	"os"

	"github.com/noviopenworks/homonto/internal/cli"
)

func main() {
	if err := cli.NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 5: Run tests + build to verify**

Run: `go test ./... && go build ./...`
Expected: PASS; binary builds.

- [ ] **Step 6: Add .gitignore and commit**

Create `.gitignore`:
```
/homonto
/.homonto/
.env
```

```bash
git add go.mod go.sum main.go internal/cli/root.go internal/cli/root_test.go .gitignore
git commit -m "feat: scaffold homonto CLI with version command"
```

---

### Task 2: Config model + TOML parser

**Files:**
- Create: `internal/config/config.go`
- Test: `internal/config/config_test.go`

**Interfaces:**
- Produces:
  - Types: `Config{ MCPs map[string]MCP; Skills Skills; Plugins Plugins; Settings Settings }`
  - `MCP{ Command []string; Env map[string]string; Targets []string }`
  - `Skills{ Own []string }`
  - `Plugins{ Claude []string; OpenCode []string }`
  - `Settings{ Claude map[string]any; OpenCode map[string]any }`
  - `Load(path string) (*Config, error)` — parse TOML file into `Config`.
  - `(MCP).TargetsOrAll() []string` — returns `Targets`, or `["claude","opencode"]` if empty.

- [ ] **Step 1: Add the TOML dependency**

Run: `go get github.com/pelletier/go-toml/v2@latest`

- [ ] **Step 2: Write the failing test**

Create `internal/config/config_test.go`:
```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

const sample = `
[mcps.codegraph]
command = ["codegraph", "serve", "--mcp"]

[mcps.brave]
command = ["npx", "-y", "server-brave"]
env = { BRAVE_API_KEY = "${pass:ai/brave}" }
targets = ["claude"]

[skills]
own = ["graphify", "comet"]

[plugins]
claude = ["claude-hud@official"]
opencode = ["@slkiser/opencode-quota"]

[settings.claude]
model = "opus"

[settings.opencode]
model = "anthropic/claude-opus-4-8"
`

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "homonto.toml")
	if err := os.WriteFile(p, []byte(sample), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := Load(p)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got := c.MCPs["codegraph"].Command; len(got) != 3 || got[0] != "codegraph" {
		t.Fatalf("codegraph command = %v", got)
	}
	if got := c.MCPs["brave"].Env["BRAVE_API_KEY"]; got != "${pass:ai/brave}" {
		t.Fatalf("brave env = %q", got)
	}
	if got := c.MCPs["codegraph"].TargetsOrAll(); len(got) != 2 {
		t.Fatalf("default targets = %v", got)
	}
	if got := c.MCPs["brave"].TargetsOrAll(); len(got) != 1 || got[0] != "claude" {
		t.Fatalf("brave targets = %v", got)
	}
	if c.Settings.Claude["model"] != "opus" {
		t.Fatalf("claude model = %v", c.Settings.Claude["model"])
	}
	if len(c.Skills.Own) != 2 {
		t.Fatalf("skills = %v", c.Skills.Own)
	}
}

func TestLoadMissingFile(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "nope.toml")); err == nil {
		t.Fatal("expected error for missing file")
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/config/`
Expected: FAIL — `undefined: Load`.

- [ ] **Step 4: Write minimal implementation**

Create `internal/config/config.go`:
```go
package config

import (
	"fmt"
	"os"

	toml "github.com/pelletier/go-toml/v2"
)

type MCP struct {
	Command []string          `toml:"command"`
	Env     map[string]string `toml:"env"`
	Targets []string          `toml:"targets"`
}

func (m MCP) TargetsOrAll() []string {
	if len(m.Targets) == 0 {
		return []string{"claude", "opencode"}
	}
	return m.Targets
}

type Skills struct {
	Own []string `toml:"own"`
}

type Plugins struct {
	Claude   []string `toml:"claude"`
	OpenCode []string `toml:"opencode"`
}

type Settings struct {
	Claude   map[string]any `toml:"claude"`
	OpenCode map[string]any `toml:"opencode"`
}

type Config struct {
	MCPs     map[string]MCP `toml:"mcps"`
	Skills   Skills         `toml:"skills"`
	Plugins  Plugins        `toml:"plugins"`
	Settings Settings       `toml:"settings"`
}

// Load reads and parses a homonto.toml file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var c Config
	if err := toml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &c, nil
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/config/`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/config/ go.mod go.sum
git commit -m "feat: config model and TOML loader"
```

---

### Task 3: Secret resolver (env + pass)

**Files:**
- Create: `internal/secret/resolver.go`
- Test: `internal/secret/resolver_test.go`

**Interfaces:**
- Produces:
  - `type Resolver struct { Getenv func(string) string; Pass func(path string) (string, error) }`
  - `NewResolver() *Resolver` — defaults `Getenv=os.Getenv`, `Pass=` real `pass show`.
  - `(*Resolver).Resolve(s string) (string, error)` — replaces every `${...}` token. `${pass:PATH}` → pass; otherwise `${NAME}` → env. Errors if a referenced value is empty/missing.
  - `ContainsRef(s string) bool` — true if `s` has a `${...}` token.

- [ ] **Step 1: Write the failing test**

Create `internal/secret/resolver_test.go`:
```go
package secret

import (
	"strings"
	"testing"
)

func TestResolveEnvAndPass(t *testing.T) {
	r := &Resolver{
		Getenv: func(k string) string {
			if k == "FOO" {
				return "envval"
			}
			return ""
		},
		Pass: func(p string) (string, error) {
			if p == "ai/brave" {
				return "passval", nil
			}
			return "", &notFound{p}
		},
	}
	got, err := r.Resolve("a=${FOO} b=${pass:ai/brave}")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got != "a=envval b=passval" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveMissingEnvErrors(t *testing.T) {
	r := &Resolver{Getenv: func(string) string { return "" }, Pass: func(string) (string, error) { return "", nil }}
	_, err := r.Resolve("${MISSING}")
	if err == nil || !strings.Contains(err.Error(), "MISSING") {
		t.Fatalf("expected missing-env error, got %v", err)
	}
}

func TestContainsRef(t *testing.T) {
	if !ContainsRef("x ${Y}") || ContainsRef("plain") {
		t.Fatal("ContainsRef wrong")
	}
}

type notFound struct{ p string }

func (e *notFound) Error() string { return "not found: " + e.p }
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/secret/`
Expected: FAIL — `undefined: Resolver`.

- [ ] **Step 3: Write minimal implementation**

Create `internal/secret/resolver.go`:
```go
package secret

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

var refRe = regexp.MustCompile(`\$\{([^}]+)\}`)

type Resolver struct {
	Getenv func(string) string
	Pass   func(path string) (string, error)
}

func NewResolver() *Resolver {
	return &Resolver{
		Getenv: os.Getenv,
		Pass: func(path string) (string, error) {
			out, err := exec.Command("pass", "show", path).Output()
			if err != nil {
				return "", fmt.Errorf("pass show %s: %w", path, err)
			}
			return strings.SplitN(strings.TrimRight(string(out), "\n"), "\n", 2)[0], nil
		},
	}
}

// ContainsRef reports whether s contains a ${...} reference.
func ContainsRef(s string) bool { return refRe.MatchString(s) }

// Resolve replaces every ${...} token in s with its resolved value.
func (r *Resolver) Resolve(s string) (string, error) {
	var firstErr error
	out := refRe.ReplaceAllStringFunc(s, func(tok string) string {
		inner := tok[2 : len(tok)-1] // strip ${ }
		var val string
		var err error
		if strings.HasPrefix(inner, "pass:") {
			val, err = r.Pass(strings.TrimPrefix(inner, "pass:"))
		} else {
			val = r.Getenv(inner)
			if val == "" {
				err = fmt.Errorf("env var %s is not set", inner)
			}
		}
		if err != nil && firstErr == nil {
			firstErr = err
		}
		return val
	})
	if firstErr != nil {
		return "", firstErr
	}
	return out, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/secret/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/secret/
git commit -m "feat: secret resolver for pass and env references"
```

---

### Task 4: State store (drift snapshot)

**Files:**
- Create: `internal/state/state.go`
- Test: `internal/state/state_test.go`

**Interfaces:**
- Produces:
  - `type State struct { Managed map[string]map[string]string }` — outer key = tool name, inner = managed-key → JSON-encoded value last applied.
  - `Load(dir string) (*State, error)` — reads `<dir>/state.json`; returns empty State (not error) if absent.
  - `(*State).Save(dir string) error` — writes `<dir>/state.json` atomically, creating `dir`.
  - `(*State).Set(tool, key, val string)` and `(*State).Get(tool, key string) (string, bool)`.

- [ ] **Step 1: Write the failing test**

Create `internal/state/state_test.go`:
```go
package state

import (
	"path/filepath"
	"testing"
)

func TestLoadAbsentReturnsEmpty(t *testing.T) {
	s, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if _, ok := s.Get("claude", "model"); ok {
		t.Fatal("expected empty state")
	}
}

func TestSaveAndReload(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested")
	s, _ := Load(dir)
	s.Set("claude", "model", `"opus"`)
	if err := s.Save(dir); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, _ := Load(dir)
	if v, ok := got.Get("claude", "model"); !ok || v != `"opus"` {
		t.Fatalf("reloaded = %q ok=%v", v, ok)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/state/`
Expected: FAIL — `undefined: Load`.

- [ ] **Step 3: Write minimal implementation**

Create `internal/state/state.go`:
```go
package state

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type State struct {
	Managed map[string]map[string]string `json:"managed"`
}

func newState() *State { return &State{Managed: map[string]map[string]string{}} }

func file(dir string) string { return filepath.Join(dir, "state.json") }

// Load reads <dir>/state.json, returning an empty State if the file is absent.
func Load(dir string) (*State, error) {
	data, err := os.ReadFile(file(dir))
	if errors.Is(err, os.ErrNotExist) {
		return newState(), nil
	}
	if err != nil {
		return nil, err
	}
	s := newState()
	if err := json.Unmarshal(data, s); err != nil {
		return nil, err
	}
	if s.Managed == nil {
		s.Managed = map[string]map[string]string{}
	}
	return s, nil
}

// Save writes the state atomically, creating dir if needed.
func (s *State) Save(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := file(dir) + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, file(dir))
}

func (s *State) Set(tool, key, val string) {
	if s.Managed[tool] == nil {
		s.Managed[tool] = map[string]string{}
	}
	s.Managed[tool][key] = val
}

func (s *State) Get(tool, key string) (string, bool) {
	v, ok := s.Managed[tool][key]
	return v, ok
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/state/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/state/
git commit -m "feat: state store for drift snapshots"
```

---

### Task 5: Surgical JSON/JSONC merge helpers

**Files:**
- Create: `internal/jsonutil/jsonutil.go`
- Test: `internal/jsonutil/jsonutil_test.go`

**Interfaces:**
- Produces:
  - `SetJSON(existing []byte, path string, value any) ([]byte, error)` — set a dotted path, preserving the rest of the document (sjson). Pretty-printed.
  - `GetJSON(existing []byte, path string) (string, bool)` — raw JSON of value at path (gjson), and whether it exists.
  - `Standardize(jsonc []byte) ([]byte, error)` — convert JSONC to plain JSON (hujson), dropping comments. If input is empty, returns `{}`.
  - `EnsureArrayElem(existing []byte, path, elem string) ([]byte, error)` — append string `elem` to the array at `path` if not already present (for plugin lists).

- [ ] **Step 1: Add deps**

Run:
```bash
go get github.com/tidwall/sjson@latest
go get github.com/tidwall/gjson@latest
go get github.com/tailscale/hujson@latest
```

- [ ] **Step 2: Write the failing test**

Create `internal/jsonutil/jsonutil_test.go`:
```go
package jsonutil

import (
	"strings"
	"testing"

	"github.com/tidwall/gjson"
)

func TestSetJSONPreservesUnmanaged(t *testing.T) {
	in := []byte(`{"keep":1,"mcpServers":{"old":{"command":["x"]}}}`)
	out, err := SetJSON(in, "mcpServers.brave", map[string]any{"command": []string{"npx"}})
	if err != nil {
		t.Fatal(err)
	}
	if gjson.GetBytes(out, "keep").Int() != 1 {
		t.Fatal("unmanaged key lost")
	}
	if gjson.GetBytes(out, "mcpServers.old.command.0").String() != "x" {
		t.Fatal("sibling lost")
	}
	if gjson.GetBytes(out, "mcpServers.brave.command.0").String() != "npx" {
		t.Fatal("new value missing")
	}
}

func TestStandardizeStripsComments(t *testing.T) {
	out, err := Standardize([]byte("{// hi\n\"a\":1,}"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "//") || gjson.GetBytes(out, "a").Int() != 1 {
		t.Fatalf("standardize wrong: %s", out)
	}
}

func TestStandardizeEmpty(t *testing.T) {
	out, _ := Standardize(nil)
	if strings.TrimSpace(string(out)) != "{}" {
		t.Fatalf("empty -> %q", out)
	}
}

func TestEnsureArrayElemIdempotent(t *testing.T) {
	out, _ := EnsureArrayElem([]byte(`{"plugin":["a"]}`), "plugin", "b")
	out, _ = EnsureArrayElem(out, "plugin", "b") // second time no-op
	if gjson.GetBytes(out, "plugin.#").Int() != 2 {
		t.Fatalf("array = %s", out)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/jsonutil/`
Expected: FAIL — `undefined: SetJSON`.

- [ ] **Step 4: Write minimal implementation**

Create `internal/jsonutil/jsonutil.go`:
```go
package jsonutil

import (
	"github.com/tailscale/hujson"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

var opts = &sjson.Options{Optimistic: false}

// SetJSON sets a dotted path to value, preserving the rest of the document.
func SetJSON(existing []byte, path string, value any) ([]byte, error) {
	if len(existing) == 0 {
		existing = []byte("{}")
	}
	out, err := sjson.SetBytesOptions(existing, path, value, opts)
	if err != nil {
		return nil, err
	}
	return pretty(out)
}

// GetJSON returns the raw JSON of the value at path and whether it exists.
func GetJSON(existing []byte, path string) (string, bool) {
	r := gjson.GetBytes(existing, path)
	if !r.Exists() {
		return "", false
	}
	return r.Raw, true
}

// Standardize converts JSONC to plain JSON (dropping comments).
func Standardize(jsonc []byte) ([]byte, error) {
	if len(jsonc) == 0 {
		return []byte("{}"), nil
	}
	v, err := hujson.Parse(jsonc)
	if err != nil {
		return nil, err
	}
	v.Standardize()
	return v.Pack(), nil
}

// EnsureArrayElem appends a string elem to the array at path if absent.
func EnsureArrayElem(existing []byte, path, elem string) ([]byte, error) {
	for _, v := range gjson.GetBytes(existing, path).Array() {
		if v.String() == elem {
			return existing, nil
		}
	}
	out, err := sjson.SetBytesOptions(existing, path+".-1", elem, opts)
	if err != nil {
		return nil, err
	}
	return pretty(out)
}

func pretty(b []byte) ([]byte, error) {
	v, err := hujson.Parse(b)
	if err != nil {
		return b, nil // already valid JSON; return as-is
	}
	v.Format()
	return v.Pack(), nil
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/jsonutil/`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/jsonutil/ go.mod go.sum
git commit -m "feat: surgical JSON/JSONC merge helpers"
```

---

### Task 6: Adapter interface, Change type, and plan printer

**Files:**
- Create: `internal/adapter/adapter.go`
- Create: `internal/plan/plan.go`
- Test: `internal/plan/plan_test.go`

**Interfaces:**
- Produces:
  - `type Change struct { Action string; Key string; Old string; New string }` where `Action ∈ {"create","update","noop"}`. `Old`/`New` may contain `${...}` tokens (unresolved). Lives in `internal/adapter`.
  - `type ChangeSet struct { Tool string; Changes []Change }` (in `internal/adapter`).
  - `type Adapter interface { Name() string; Plan(c *config.Config, st *state.State) (ChangeSet, error); Apply(cs ChangeSet, res *secret.Resolver, st *state.State) error }` (in `internal/adapter`).
  - `plan.Render(sets []adapter.ChangeSet) string` — terraform-style text; `+`=create, `~`=update; omits noops; never resolves secrets.
  - `plan.HasChanges(sets []adapter.ChangeSet) bool`.

- [ ] **Step 1: Write the failing test**

Create `internal/plan/plan_test.go`:
```go
package plan

import (
	"strings"
	"testing"

	"github.com/noviopenworks/homonto/internal/adapter"
)

func TestRenderShowsChangesNotNoops(t *testing.T) {
	sets := []adapter.ChangeSet{{
		Tool: "claude",
		Changes: []adapter.Change{
			{Action: "update", Key: "settings.model", Old: `"sonnet"`, New: `"opus"`},
			{Action: "create", Key: "mcp.brave", New: `{"command":["npx"]}`},
			{Action: "noop", Key: "mcp.codegraph"},
		},
	}}
	out := Render(sets)
	if !strings.Contains(out, "~ settings.model") || !strings.Contains(out, `"sonnet" -> "opus"`) {
		t.Fatalf("update line missing:\n%s", out)
	}
	if !strings.Contains(out, "+ mcp.brave") {
		t.Fatalf("create line missing:\n%s", out)
	}
	if strings.Contains(out, "codegraph") {
		t.Fatalf("noop should be hidden:\n%s", out)
	}
	if !HasChanges(sets) {
		t.Fatal("HasChanges should be true")
	}
}

func TestRenderNeverResolvesSecrets(t *testing.T) {
	sets := []adapter.ChangeSet{{Tool: "claude", Changes: []adapter.Change{
		{Action: "create", Key: "mcp.brave.env", New: `{"BRAVE_API_KEY":"${pass:ai/brave}"}`},
	}}}
	if !strings.Contains(Render(sets), "${pass:ai/brave}") {
		t.Fatal("plan must show the unresolved token verbatim")
	}
}

func TestHasChangesFalseWhenAllNoop(t *testing.T) {
	sets := []adapter.ChangeSet{{Tool: "claude", Changes: []adapter.Change{{Action: "noop", Key: "x"}}}}
	if HasChanges(sets) {
		t.Fatal("expected no changes")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/plan/`
Expected: FAIL — `undefined: adapter` package / `Render`.

- [ ] **Step 3: Write the adapter types**

Create `internal/adapter/adapter.go`:
```go
package adapter

import (
	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/secret"
	"github.com/noviopenworks/homonto/internal/state"
)

// Change is a single planned modification. Old/New hold JSON-encoded values
// and may still contain unresolved ${...} secret tokens.
type Change struct {
	Action string // "create" | "update" | "noop"
	Key    string
	Old    string
	New    string
}

type ChangeSet struct {
	Tool    string
	Changes []Change
}

// Adapter projects desired config into one target tool.
type Adapter interface {
	Name() string
	Plan(c *config.Config, st *state.State) (ChangeSet, error)
	Apply(cs ChangeSet, res *secret.Resolver, st *state.State) error
}
```

- [ ] **Step 4: Write the plan printer**

Create `internal/plan/plan.go`:
```go
package plan

import (
	"fmt"
	"strings"

	"github.com/noviopenworks/homonto/internal/adapter"
)

// HasChanges reports whether any change is not a noop.
func HasChanges(sets []adapter.ChangeSet) bool {
	for _, s := range sets {
		for _, c := range s.Changes {
			if c.Action != "noop" {
				return true
			}
		}
	}
	return false
}

// Render produces a terraform-style plan. It never resolves secrets.
func Render(sets []adapter.ChangeSet) string {
	var b strings.Builder
	for _, s := range sets {
		var lines []string
		for _, c := range s.Changes {
			switch c.Action {
			case "create":
				lines = append(lines, fmt.Sprintf("  + %s = %s", c.Key, c.New))
			case "update":
				lines = append(lines, fmt.Sprintf("  ~ %s: %s -> %s", c.Key, c.Old, c.New))
			}
		}
		if len(lines) == 0 {
			continue
		}
		fmt.Fprintf(&b, "%s:\n%s\n", s.Tool, strings.Join(lines, "\n"))
	}
	return b.String()
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/plan/ ./internal/adapter/`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/adapter/ internal/plan/
git commit -m "feat: adapter interface, change type, and plan printer"
```

---

### Task 7: Content linker (symlinks + conflict detection)

**Files:**
- Create: `internal/link/linker.go`
- Test: `internal/link/linker_test.go`

**Interfaces:**
- Produces:
  - `Link(src, dst string) (changed bool, err error)` — ensure `dst` is a symlink to `src`. Creates parent dirs. Returns `changed=false` if already correct. Errors (does not clobber) if `dst` exists and is not a symlink, or points elsewhere — error message contains `"conflict"`.
  - `LinkPlan(srcs map[string]string) ([]string, error)` — given dst→src map, return human descriptions of links that would change (for plan output); pure check, no writes.

- [ ] **Step 1: Write the failing test**

Create `internal/link/linker_test.go`:
```go
package link

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLinkCreatesAndIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "content", "skills", "graphify")
	os.MkdirAll(src, 0o755)
	dst := filepath.Join(dir, "claude", "skills", "graphify")

	changed, err := Link(src, dst)
	if err != nil || !changed {
		t.Fatalf("first link changed=%v err=%v", changed, err)
	}
	got, _ := os.Readlink(dst)
	if got != src {
		t.Fatalf("symlink points to %q", got)
	}
	changed, err = Link(src, dst)
	if err != nil || changed {
		t.Fatalf("second link should be no-op: changed=%v err=%v", changed, err)
	}
}

func TestLinkConflictDoesNotClobber(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	os.MkdirAll(src, 0o755)
	dst := filepath.Join(dir, "dst")
	os.WriteFile(dst, []byte("real file"), 0o644) // not a symlink

	_, err := Link(src, dst)
	if err == nil || !strings.Contains(err.Error(), "conflict") {
		t.Fatalf("expected conflict error, got %v", err)
	}
	if b, _ := os.ReadFile(dst); string(b) != "real file" {
		t.Fatal("conflict clobbered the real file")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/link/`
Expected: FAIL — `undefined: Link`.

- [ ] **Step 3: Write minimal implementation**

Create `internal/link/linker.go`:
```go
package link

import (
	"fmt"
	"os"
	"path/filepath"
)

// Link ensures dst is a symlink to src, returning whether it changed.
func Link(src, dst string) (bool, error) {
	if fi, err := os.Lstat(dst); err == nil {
		if fi.Mode()&os.ModeSymlink == 0 {
			return false, fmt.Errorf("conflict: %s exists and is not a symlink", dst)
		}
		cur, _ := os.Readlink(dst)
		if cur == src {
			return false, nil
		}
		return false, fmt.Errorf("conflict: %s links to %s, not %s", dst, cur, src)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return false, err
	}
	if err := os.Symlink(src, dst); err != nil {
		return false, err
	}
	return true, nil
}

// LinkPlan returns descriptions of links (dst->src) that would change.
func LinkPlan(srcs map[string]string) ([]string, error) {
	var out []string
	for dst, src := range srcs {
		fi, err := os.Lstat(dst)
		if err != nil {
			out = append(out, fmt.Sprintf("+ link %s -> %s", dst, src))
			continue
		}
		if fi.Mode()&os.ModeSymlink == 0 {
			return nil, fmt.Errorf("conflict: %s exists and is not a symlink", dst)
		}
		if cur, _ := os.Readlink(dst); cur != src {
			out = append(out, fmt.Sprintf("~ relink %s -> %s", dst, src))
		}
	}
	return out, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/link/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/link/
git commit -m "feat: content linker with conflict detection"
```

---

### Task 8: Claude Code adapter

**Files:**
- Create: `internal/adapter/claude/claude.go`
- Test: `internal/adapter/claude/claude_test.go`

**Interfaces:**
- Consumes: `config.Config`, `state.State`, `secret.Resolver`, `jsonutil.*`, `adapter.Change/ChangeSet`.
- Produces:
  - `New(home string) *Adapter` — `home` is the `$HOME` root (so tests inject a temp dir). Files: `home/.claude.json` (mcpServers), `home/.claude/settings.json` (settings + plugins), skill symlinks under `home/.claude/skills/`.
  - Implements `adapter.Adapter`: `Name()=="claude"`.
  - Managed keys use the form `mcp.<name>`, `setting.<key>`, `plugin.<name>` in plan output and state.

- [ ] **Step 1: Write the failing test (MCP + settings projection, surgical)**

Create `internal/adapter/claude/claude_test.go`:
```go
package claude

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/secret"
	"github.com/noviopenworks/homonto/internal/state"
	"github.com/tidwall/gjson"
)

func cfg() *config.Config {
	return &config.Config{
		MCPs: map[string]config.MCP{
			"brave": {Command: []string{"npx", "server-brave"}, Env: map[string]string{"K": "${pass:ai/brave}"}, Targets: []string{"claude"}},
		},
		Settings: config.Settings{Claude: map[string]any{"model": "opus"}},
	}
}

func TestPlanThenApplyIsSurgicalAndIdempotent(t *testing.T) {
	home := t.TempDir()
	// pre-existing unmanaged content must survive
	os.WriteFile(filepath.Join(home, ".claude.json"), []byte(`{"keep":true,"mcpServers":{}}`), 0o644)
	os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	os.WriteFile(filepath.Join(home, ".claude", "settings.json"), []byte(`{"theme":"dark"}`), 0o644)

	a := New(home)
	st := stateEmpty()
	res := &secret.Resolver{
		Getenv: os.Getenv,
		Pass:   func(string) (string, error) { return "SECRET", nil },
	}

	cs, err := a.Plan(cfg(), st)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if len(cs.Changes) == 0 {
		t.Fatal("expected changes on first plan")
	}
	if err := a.Apply(cs, res, st); err != nil {
		t.Fatalf("apply: %v", err)
	}

	mj, _ := os.ReadFile(filepath.Join(home, ".claude.json"))
	if !gjson.GetBytes(mj, "keep").Bool() {
		t.Fatal("unmanaged .claude.json key lost")
	}
	if gjson.GetBytes(mj, "mcpServers.brave.env.K").String() != "SECRET" {
		t.Fatalf("secret not resolved on apply: %s", mj)
	}
	sj, _ := os.ReadFile(filepath.Join(home, ".claude", "settings.json"))
	if gjson.GetBytes(sj, "theme").String() != "dark" {
		t.Fatal("unmanaged settings key lost")
	}
	if gjson.GetBytes(sj, "model").String() != "opus" {
		t.Fatal("managed setting not written")
	}

	// second plan = no changes (idempotent)
	cs2, _ := a.Plan(cfg(), st)
	for _, c := range cs2.Changes {
		if c.Action != "noop" {
			t.Fatalf("expected idempotent noop, got %+v", c)
		}
	}
}

func stateEmpty() *state.State { s, _ := state.Load(filepathJoinTmp()); return s }
func filepathJoinTmp() string  { d, _ := os.MkdirTemp("", "st"); return d }
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/adapter/claude/`
Expected: FAIL — `undefined: New`.

- [ ] **Step 3: Write minimal implementation**

Create `internal/adapter/claude/claude.go`:
```go
package claude

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/noviopenworks/homonto/internal/adapter"
	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/jsonutil"
	"github.com/noviopenworks/homonto/internal/secret"
	"github.com/noviopenworks/homonto/internal/state"
)

type Adapter struct{ home string }

func New(home string) *Adapter { return &Adapter{home: home} }

func (a *Adapter) Name() string { return "claude" }

func (a *Adapter) claudeJSON() string   { return filepath.Join(a.home, ".claude.json") }
func (a *Adapter) settingsJSON() string { return filepath.Join(a.home, ".claude", "settings.json") }

// desired returns managed key -> JSON-encoded desired value (with unresolved tokens).
func (a *Adapter) desired(c *config.Config) map[string]string {
	out := map[string]string{}
	for name, m := range c.MCPs {
		if !contains(m.TargetsOrAll(), "claude") {
			continue
		}
		obj := map[string]any{"command": m.Command}
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
	cur, err := a.current()
	if err != nil {
		return adapter.ChangeSet{}, err
	}
	cs := adapter.ChangeSet{Tool: "claude"}
	for key, want := range a.desired(c) {
		old, ok := cur[key]
		switch {
		case !ok:
			cs.Changes = append(cs.Changes, adapter.Change{Action: "create", Key: key, New: want})
		case old != want:
			cs.Changes = append(cs.Changes, adapter.Change{Action: "update", Key: key, Old: old, New: want})
		default:
			cs.Changes = append(cs.Changes, adapter.Change{Action: "noop", Key: key})
		}
	}
	return cs, nil
}

// current reads existing managed values from disk, keyed like desired().
func (a *Adapter) current() (map[string]string, error) {
	out := map[string]string{}
	mj, err := readFileOrEmpty(a.claudeJSON())
	if err != nil {
		return nil, err
	}
	sj, err := readFileOrEmpty(a.settingsJSON())
	if err != nil {
		return nil, err
	}
	mj, _ = jsonutil.Standardize(mj)
	sj, _ = jsonutil.Standardize(sj)
	// We re-derive keys lazily in Apply; for Plan we only need values that exist.
	a.collect(out, mj, "mcpServers", "mcp.")
	a.collectSettings(out, sj)
	a.collect(out, sj, "enabledPlugins", "plugin.")
	return out, nil
}

func (a *Adapter) collect(out map[string]string, doc []byte, root, prefix string) {
	for k, v := range objMembers(doc, root) {
		out[prefix+k] = v
	}
}

func (a *Adapter) collectSettings(out map[string]string, doc []byte) {
	var m map[string]json.RawMessage
	_ = json.Unmarshal(doc, &m)
	for k, raw := range m {
		if k == "mcpServers" || k == "enabledPlugins" {
			continue
		}
		out["setting."+k] = string(raw)
	}
}

func (a *Adapter) Apply(cs adapter.ChangeSet, res *secret.Resolver, st *state.State) error {
	// Read current docs.
	mj, err := readFileOrEmpty(a.claudeJSON())
	if err != nil {
		return err
	}
	sj, err := readFileOrEmpty(a.settingsJSON())
	if err != nil {
		return err
	}
	mj, _ = jsonutil.Standardize(mj)
	sj, _ = jsonutil.Standardize(sj)

	for _, c := range cs.Changes {
		if c.Action == "noop" {
			continue
		}
		resolved, err := res.Resolve(c.New)
		if err != nil {
			return err
		}
		var val any
		if err := json.Unmarshal([]byte(resolved), &val); err != nil {
			return err
		}
		switch {
		case hasPrefix(c.Key, "mcp."):
			mj, err = jsonutil.SetJSON(mj, "mcpServers."+trim(c.Key, "mcp."), val)
		case hasPrefix(c.Key, "setting."):
			sj, err = jsonutil.SetJSON(sj, trim(c.Key, "setting."), val)
		case hasPrefix(c.Key, "plugin."):
			sj, err = jsonutil.SetJSON(sj, "enabledPlugins."+trim(c.Key, "plugin."), val)
		}
		if err != nil {
			return err
		}
		st.Set("claude", c.Key, c.New) // store the unresolved form
	}
	if err := writeAtomic(a.claudeJSON(), mj); err != nil {
		return err
	}
	return writeAtomic(a.settingsJSON(), sj)
}
```

Create `internal/adapter/claude/util.go`:
```go
package claude

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/tidwall/gjson"
)

func contains(ss []string, x string) bool {
	for _, s := range ss {
		if s == x {
			return true
		}
	}
	return false
}

func mustJSON(v any) string { b, _ := json.Marshal(v); return string(b) }

func hasPrefix(s, p string) bool { return strings.HasPrefix(s, p) }
func trim(s, p string) string    { return strings.TrimPrefix(s, p) }

func readFileOrEmpty(path string) ([]byte, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return b, err
}

func objMembers(doc []byte, root string) map[string]string {
	out := map[string]string{}
	gjson.GetBytes(doc, root).ForEach(func(k, v gjson.Result) bool {
		out[k.String()] = v.Raw
		return true
	})
	return out
}

func writeAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/adapter/claude/`
Expected: PASS. If the idempotency assertion fails because JSON value formatting differs (e.g. `"opus"` vs `opus`), normalize both sides through `json.Marshal` of the unmarshaled value in `current()` before comparing — adjust `collect`/`collectSettings` to re-marshal `v.Value()`.

- [ ] **Step 5: Commit**

```bash
git add internal/adapter/claude/
git commit -m "feat: Claude Code adapter (mcp, settings, plugins)"
```

---

### Task 9: Claude skill linking + OpenCode adapter

**Files:**
- Modify: `internal/adapter/claude/claude.go` (add skill links to `Apply`)
- Create: `internal/adapter/opencode/opencode.go`
- Create: `internal/adapter/opencode/util.go`
- Test: `internal/adapter/opencode/opencode_test.go`

**Interfaces:**
- Consumes: same as Task 8, plus `link.Link`, and a `contentDir` for owned skills.
- Produces:
  - `claude.New` gains a second arg: `New(home, contentDir string)`. In `Apply`, for each `c.Skills.Own`, link `contentDir/skills/<n>` → `home/.claude/skills/<n>`. (Update Task 8 callers/tests accordingly — the test's `New(home)` becomes `New(home, t.TempDir())`.)
  - `opencode.New(home, contentDir string) *Adapter`, `Name()=="opencode"`. MCP/settings/plugins live in `home/.config/opencode/opencode.jsonc`; skills symlink under `home/.config/opencode/skills/`. MCP object shape: `{"type":"local","command":[...],"enabled":true}` (+ `environment` map if env set). Plugins append to the `plugin` array via `jsonutil.EnsureArrayElem`.

- [ ] **Step 1: Update Claude adapter signature + skill links**

In `internal/adapter/claude/claude.go`, change:
```go
type Adapter struct{ home, content string }

func New(home, content string) *Adapter { return &Adapter{home: home, content: content} }
```
At the end of `Apply` (before the final `writeAtomic` of settings), add skill linking:
```go
	for _, name := range a.skills {
		src := filepath.Join(a.content, "skills", name)
		dst := filepath.Join(a.home, ".claude", "skills", name)
		if _, err := link.Link(src, dst); err != nil {
			return err
		}
	}
```
Add `a.skills []string` populated in `Plan` from `c.Skills.Own`, and import `github.com/noviopenworks/homonto/internal/link`. Update `claude_test.go` call site to `New(home, t.TempDir())`.

Run: `go test ./internal/adapter/claude/`
Expected: PASS (after updating the test's `New` call).

- [ ] **Step 2: Write the failing OpenCode test**

Create `internal/adapter/opencode/opencode_test.go`:
```go
package opencode

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/secret"
	"github.com/noviopenworks/homonto/internal/state"
	"github.com/noviopenworks/homonto/internal/jsonutil"
	"github.com/tidwall/gjson"
)

func TestOpenCodeProjectsMCPAndPreservesKeys(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".config", "opencode")
	os.MkdirAll(dir, 0o755)
	// JSONC with a comment + unmanaged key
	os.WriteFile(filepath.Join(dir, "opencode.jsonc"), []byte("{\n  // keep me\n  \"theme\":\"x\",\n  \"plugin\":[\"existing\"]\n}"), 0o644)

	a := New(home, t.TempDir())
	st, _ := state.Load(t.TempDir())
	c := &config.Config{
		MCPs:    map[string]config.MCP{"codegraph": {Command: []string{"codegraph", "serve"}, Targets: []string{"opencode"}}},
		Plugins: config.Plugins{OpenCode: []string{"@slkiser/opencode-quota"}},
	}
	res := &secret.Resolver{Getenv: os.Getenv, Pass: func(string) (string, error) { return "", nil }}

	cs, err := a.Plan(c, st)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Apply(cs, res, st); err != nil {
		t.Fatal(err)
	}

	raw, _ := os.ReadFile(filepath.Join(dir, "opencode.jsonc"))
	doc, _ := jsonutil.Standardize(raw)
	if gjson.GetBytes(doc, "theme").String() != "x" {
		t.Fatal("unmanaged key lost")
	}
	if gjson.GetBytes(doc, "mcp.codegraph.type").String() != "local" {
		t.Fatalf("mcp not projected: %s", doc)
	}
	if gjson.GetBytes(doc, "mcp.codegraph.command.0").String() != "codegraph" {
		t.Fatal("mcp command missing")
	}
	plugins := gjson.GetBytes(doc, "plugin").Array()
	if len(plugins) != 2 {
		t.Fatalf("plugin array = %v", plugins)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/adapter/opencode/`
Expected: FAIL — `undefined: New`.

- [ ] **Step 4: Write minimal implementation**

Create `internal/adapter/opencode/opencode.go`:
```go
package opencode

import (
	"encoding/json"
	"path/filepath"

	"github.com/noviopenworks/homonto/internal/adapter"
	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/jsonutil"
	"github.com/noviopenworks/homonto/internal/link"
	"github.com/noviopenworks/homonto/internal/secret"
	"github.com/noviopenworks/homonto/internal/state"
)

type Adapter struct {
	home, content string
	skills        []string
}

func New(home, content string) *Adapter { return &Adapter{home: home, content: content} }

func (a *Adapter) Name() string { return "opencode" }
func (a *Adapter) cfgFile() string {
	return filepath.Join(a.home, ".config", "opencode", "opencode.jsonc")
}

func (a *Adapter) desired(c *config.Config) (mcps map[string]string, plugins []string) {
	mcps = map[string]string{}
	for name, m := range c.MCPs {
		if !contains(m.TargetsOrAll(), "opencode") {
			continue
		}
		obj := map[string]any{"type": "local", "command": m.Command, "enabled": true}
		if len(m.Env) > 0 {
			obj["environment"] = m.Env
		}
		mcps[name] = mustJSON(obj)
	}
	return mcps, c.Plugins.OpenCode
}

func (a *Adapter) Plan(c *config.Config, st *state.State) (adapter.ChangeSet, error) {
	a.skills = c.Skills.Own
	doc, err := readStandardized(a.cfgFile())
	if err != nil {
		return adapter.ChangeSet{}, err
	}
	mcps, plugins := a.desired(c)
	cs := adapter.ChangeSet{Tool: "opencode"}
	for name, want := range mcps {
		key := "mcp." + name
		if old, ok := jsonutil.GetJSON(doc, "mcp."+name); !ok {
			cs.Changes = append(cs.Changes, adapter.Change{Action: "create", Key: key, New: want})
		} else if !jsonEqual(old, want) {
			cs.Changes = append(cs.Changes, adapter.Change{Action: "update", Key: key, Old: old, New: want})
		} else {
			cs.Changes = append(cs.Changes, adapter.Change{Action: "noop", Key: key})
		}
	}
	for k, v := range c.Settings.OpenCode {
		key := "setting." + k
		want := mustJSON(v)
		if old, ok := jsonutil.GetJSON(doc, k); !ok {
			cs.Changes = append(cs.Changes, adapter.Change{Action: "create", Key: key, New: want})
		} else if !jsonEqual(old, want) {
			cs.Changes = append(cs.Changes, adapter.Change{Action: "update", Key: key, Old: old, New: want})
		} else {
			cs.Changes = append(cs.Changes, adapter.Change{Action: "noop", Key: key})
		}
	}
	for _, p := range plugins {
		if !arrayHas(doc, "plugin", p) {
			cs.Changes = append(cs.Changes, adapter.Change{Action: "create", Key: "plugin." + p, New: mustJSON(p)})
		} else {
			cs.Changes = append(cs.Changes, adapter.Change{Action: "noop", Key: "plugin." + p})
		}
	}
	return cs, nil
}

func (a *Adapter) Apply(cs adapter.ChangeSet, res *secret.Resolver, st *state.State) error {
	doc, err := readStandardized(a.cfgFile())
	if err != nil {
		return err
	}
	for _, c := range cs.Changes {
		if c.Action == "noop" {
			continue
		}
		resolved, err := res.Resolve(c.New)
		if err != nil {
			return err
		}
		switch {
		case hasPrefix(c.Key, "mcp."):
			var val any
			if err := json.Unmarshal([]byte(resolved), &val); err != nil {
				return err
			}
			doc, err = jsonutil.SetJSON(doc, "mcp."+trim(c.Key, "mcp."), val)
		case hasPrefix(c.Key, "setting."):
			var val any
			if err := json.Unmarshal([]byte(resolved), &val); err != nil {
				return err
			}
			doc, err = jsonutil.SetJSON(doc, trim(c.Key, "setting."), val)
		case hasPrefix(c.Key, "plugin."):
			doc, err = jsonutil.EnsureArrayElem(doc, "plugin", trim(c.Key, "plugin."))
		}
		if err != nil {
			return err
		}
		st.Set("opencode", c.Key, c.New)
	}
	if err := writeAtomic(a.cfgFile(), doc); err != nil {
		return err
	}
	for _, name := range a.skills {
		src := filepath.Join(a.content, "skills", name)
		dst := filepath.Join(a.home, ".config", "opencode", "skills", name)
		if _, err := link.Link(src, dst); err != nil {
			return err
		}
	}
	return nil
}
```

Create `internal/adapter/opencode/util.go`:
```go
package opencode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/noviopenworks/homonto/internal/jsonutil"
	"github.com/tidwall/gjson"
)

func contains(ss []string, x string) bool {
	for _, s := range ss {
		if s == x {
			return true
		}
	}
	return false
}

func arrayHas(doc []byte, path, elem string) bool {
	for _, v := range gjson.GetBytes(doc, path).Array() {
		if v.String() == elem {
			return true
		}
	}
	return false
}

func mustJSON(v any) string    { b, _ := json.Marshal(v); return string(b) }
func hasPrefix(s, p string) bool { return strings.HasPrefix(s, p) }
func trim(s, p string) string    { return strings.TrimPrefix(s, p) }

// jsonEqual compares two JSON strings structurally.
func jsonEqual(a, b string) bool {
	var x, y any
	if json.Unmarshal([]byte(a), &x) != nil || json.Unmarshal([]byte(b), &y) != nil {
		return a == b
	}
	bx, _ := json.Marshal(x)
	by, _ := json.Marshal(y)
	return string(bx) == string(by)
}

func readStandardized(path string) ([]byte, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return jsonutil.Standardize(nil)
	}
	if err != nil {
		return nil, err
	}
	return jsonutil.Standardize(b)
}

func writeAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/adapter/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/adapter/
git commit -m "feat: OpenCode adapter + Claude skill linking"
```

---

### Task 10: Engine + `plan` and `apply` commands

**Files:**
- Create: `internal/engine/engine.go`
- Create: `internal/cli/plan.go`
- Create: `internal/cli/apply.go`
- Test: `internal/engine/engine_test.go`

**Interfaces:**
- Consumes: `config`, all adapters, `plan`, `secret`, `state`.
- Produces:
  - `type Engine struct { Cfg *config.Config; Adapters []adapter.Adapter; State *state.State; StateDir string; Resolver *secret.Resolver }`
  - `Build(configPath, home, contentDir string) (*Engine, error)` — loads config, builds both adapters with `home`+`contentDir`, loads state from `<repoDir>/.homonto`.
  - `(*Engine).Plan() ([]adapter.ChangeSet, error)` — runs each adapter's `Plan`.
  - `(*Engine).Apply(sets []adapter.ChangeSet) error` — two-phase: pre-resolve every non-noop change's secrets (abort on any error before writing), then call each adapter's `Apply`, then `State.Save`.
  - CLI: `homonto plan` prints `plan.Render` or "No changes."; `homonto apply` prints plan, prompts `[y/N]` (skipped with `--yes`), then applies.

- [ ] **Step 1: Write the failing engine test (two-phase abort)**

Create `internal/engine/engine_test.go`:
```go
package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/noviopenworks/homonto/internal/secret"
)

const cfgTOML = `
[mcps.brave]
command = ["npx","server-brave"]
env = { K = "${MISSING_VAR}" }
targets = ["claude"]
`

func TestApplyAbortsBeforeWritingOnMissingSecret(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	cfgPath := filepath.Join(repo, "homonto.toml")
	os.WriteFile(cfgPath, []byte(cfgTOML), 0o644)

	e, err := Build(cfgPath, home, filepath.Join(repo, "content"))
	if err != nil {
		t.Fatal(err)
	}
	e.StateDir = filepath.Join(repo, ".homonto")
	e.Resolver = &secret.Resolver{Getenv: func(string) string { return "" }, Pass: func(string) (string, error) { return "", nil }}

	sets, err := e.Plan()
	if err != nil {
		t.Fatal(err)
	}
	if err := e.Apply(sets); err == nil {
		t.Fatal("expected apply to fail on missing secret")
	}
	// No file should have been written.
	if _, err := os.Stat(filepath.Join(home, ".claude.json")); !os.IsNotExist(err) {
		t.Fatal("apply wrote a file despite secret failure (not two-phase)")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/engine/`
Expected: FAIL — `undefined: Build`.

- [ ] **Step 3: Write minimal implementation**

Create `internal/engine/engine.go`:
```go
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

type Engine struct {
	Cfg      *config.Config
	Adapters []adapter.Adapter
	State    *state.State
	StateDir string
	Resolver *secret.Resolver
}

// Build wires the engine. home is $HOME; contentDir holds owned content;
// state lives next to the config in <repo>/.homonto.
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
		Cfg:      cfg,
		Adapters: []adapter.Adapter{claude.New(home, contentDir), opencode.New(home, contentDir)},
		State:    st,
		StateDir: stateDir,
		Resolver: secret.NewResolver(),
	}, nil
}

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

func (e *Engine) Apply(sets []adapter.ChangeSet) error {
	// Phase 1: resolve ALL secrets first; abort before any write.
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
	// Phase 2: write.
	for i, a := range e.Adapters {
		if err := a.Apply(sets[i], e.Resolver, e.State); err != nil {
			return err
		}
	}
	return e.State.Save(e.StateDir)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/engine/`
Expected: PASS.

- [ ] **Step 5: Wire the CLI commands**

Create `internal/cli/plan.go`:
```go
package cli

import (
	"os"

	"github.com/noviopenworks/homonto/internal/engine"
	"github.com/noviopenworks/homonto/internal/plan"
	"github.com/spf13/cobra"
)

func planCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "plan",
		Short: "Show what apply would change",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfgPath, _ := cmd.Flags().GetString("config")
			home, _ := os.UserHomeDir()
			e, err := engine.Build(cfgPath, home, "content")
			if err != nil {
				return err
			}
			sets, err := e.Plan()
			if err != nil {
				return err
			}
			if !plan.HasChanges(sets) {
				cmd.Println("No changes. Everything up to date.")
				return nil
			}
			cmd.Print(plan.Render(sets))
			return nil
		},
	}
}
```

Create `internal/cli/apply.go`:
```go
package cli

import (
	"bufio"
	"os"
	"strings"

	"github.com/noviopenworks/homonto/internal/engine"
	"github.com/noviopenworks/homonto/internal/plan"
	"github.com/spf13/cobra"
)

func applyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Project config into the AI tools",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfgPath, _ := cmd.Flags().GetString("config")
			yes, _ := cmd.Flags().GetBool("yes")
			home, _ := os.UserHomeDir()
			e, err := engine.Build(cfgPath, home, "content")
			if err != nil {
				return err
			}
			sets, err := e.Plan()
			if err != nil {
				return err
			}
			if !plan.HasChanges(sets) {
				cmd.Println("No changes. Everything up to date.")
				return nil
			}
			cmd.Print(plan.Render(sets))
			if !yes {
				cmd.Print("\nApply these changes? [y/N] ")
				r := bufio.NewReader(os.Stdin)
				line, _ := r.ReadString('\n')
				if strings.ToLower(strings.TrimSpace(line)) != "y" {
					cmd.Println("Aborted.")
					return nil
				}
			}
			if err := e.Apply(sets); err != nil {
				return err
			}
			cmd.Println("Applied.")
			return nil
		},
	}
	cmd.Flags().Bool("yes", false, "skip confirmation")
	return cmd
}
```

Register both in `internal/cli/root.go` (`NewRootCmd`), after the version command:
```go
	root.AddCommand(planCmd(), applyCmd())
```

- [ ] **Step 6: Run all tests + build**

Run: `go test ./... && go build ./...`
Expected: PASS; builds.

- [ ] **Step 7: Commit**

```bash
git add internal/engine/ internal/cli/plan.go internal/cli/apply.go internal/cli/root.go
git commit -m "feat: engine with two-phase apply, plan and apply commands"
```

---

### Task 11: `status` (drift) and `doctor` commands

**Files:**
- Create: `internal/engine/status.go`
- Create: `internal/cli/status.go`
- Create: `internal/cli/doctor.go`
- Test: `internal/engine/status_test.go`

**Interfaces:**
- Consumes: `Engine`, `state`, `secret`.
- Produces:
  - `(*Engine).Drift() ([]string, error)` — for each tool/key in `State`, compare the value currently on disk to the snapshot; return human lines for any mismatch (`claude setting.model drifted`). Re-uses each adapter's `Plan` (an update/create where state said noop == drift).
  - `(*Engine).Doctor() []string` — checks: `pass` on PATH? each tool's config dir present? each owned skill present in `content/skills`? Returns status lines (`ok:`/`warn:`).
  - CLI `homonto status` prints drift (or "No drift."); `homonto doctor` prints checks.

- [ ] **Step 1: Write the failing test**

Create `internal/engine/status_test.go`:
```go
package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctorFlagsMissingSkillContent(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	os.WriteFile(filepath.Join(repo, "homonto.toml"), []byte("[skills]\nown=[\"ghost\"]\n"), 0o644)

	e, err := Build(filepath.Join(repo, "homonto.toml"), home, filepath.Join(repo, "content"))
	if err != nil {
		t.Fatal(err)
	}
	e.ContentDir = filepath.Join(repo, "content")
	lines := strings.Join(e.Doctor(), "\n")
	if !strings.Contains(lines, "ghost") {
		t.Fatalf("doctor should flag missing skill 'ghost':\n%s", lines)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/engine/ -run Doctor`
Expected: FAIL — `e.ContentDir undefined` / `e.Doctor undefined`.

- [ ] **Step 3: Write minimal implementation**

Add `ContentDir string` field to `Engine` in `engine.go`, and set it in `Build` (`ContentDir: contentDir`).

Create `internal/engine/status.go`:
```go
package engine

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Drift returns lines describing on-disk values that diverge from the snapshot.
func (e *Engine) Drift() ([]string, error) {
	sets, err := e.Plan()
	if err != nil {
		return nil, err
	}
	var lines []string
	for _, cs := range sets {
		for _, c := range cs.Changes {
			if c.Action == "update" {
				if _, ok := e.State.Get(cs.Tool, c.Key); ok {
					lines = append(lines, fmt.Sprintf("%s %s drifted: on-disk %s, expected %s", cs.Tool, c.Key, c.Old, c.New))
				}
			}
		}
	}
	return lines, nil
}

// Doctor runs environment health checks.
func (e *Engine) Doctor() []string {
	var out []string
	if _, err := exec.LookPath("pass"); err != nil {
		out = append(out, "warn: `pass` not found on PATH (pass: references will fail)")
	} else {
		out = append(out, "ok: pass found")
	}
	for _, name := range e.Cfg.Skills.Own {
		p := filepath.Join(e.ContentDir, "skills", name)
		if _, err := os.Stat(p); err != nil {
			out = append(out, fmt.Sprintf("warn: skill %q missing from %s", name, p))
		} else {
			out = append(out, fmt.Sprintf("ok: skill %q present", name))
		}
	}
	return out
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/engine/ -run Doctor`
Expected: PASS.

- [ ] **Step 5: Wire CLI commands**

Create `internal/cli/status.go`:
```go
package cli

import (
	"os"

	"github.com/noviopenworks/homonto/internal/engine"
	"github.com/spf13/cobra"
)

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show config drift since last apply",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfgPath, _ := cmd.Flags().GetString("config")
			home, _ := os.UserHomeDir()
			e, err := engine.Build(cfgPath, home, "content")
			if err != nil {
				return err
			}
			lines, err := e.Drift()
			if err != nil {
				return err
			}
			if len(lines) == 0 {
				cmd.Println("No drift.")
				return nil
			}
			for _, l := range lines {
				cmd.Println(l)
			}
			return nil
		},
	}
}
```

Create `internal/cli/doctor.go`:
```go
package cli

import (
	"os"

	"github.com/noviopenworks/homonto/internal/engine"
	"github.com/spf13/cobra"
)

func doctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check environment health",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfgPath, _ := cmd.Flags().GetString("config")
			home, _ := os.UserHomeDir()
			e, err := engine.Build(cfgPath, home, "content")
			if err != nil {
				return err
			}
			for _, l := range e.Doctor() {
				cmd.Println(l)
			}
			return nil
		},
	}
}
```

Register in `NewRootCmd`: `root.AddCommand(planCmd(), applyCmd(), statusCmd(), doctorCmd())`.

- [ ] **Step 6: Run all tests + build, commit**

Run: `go test ./... && go build ./...`
Expected: PASS.
```bash
git add internal/engine/ internal/cli/status.go internal/cli/doctor.go internal/cli/root.go
git commit -m "feat: status (drift) and doctor commands"
```

---

### Task 12: `init` command (scaffold a repo)

**Files:**
- Create: `internal/scaffold/scaffold.go`
- Create: `internal/cli/init.go`
- Test: `internal/scaffold/scaffold_test.go`

**Interfaces:**
- Produces:
  - `Init(dir string) (created []string, err error)` — writes (only if absent) `homonto.toml` (commented starter), `.gitignore`, `.env.example`, and `content/skills/.gitkeep`. Returns the list of created paths. Never overwrites an existing file.
  - CLI `homonto init [dir]` (default `.`).

- [ ] **Step 1: Write the failing test**

Create `internal/scaffold/scaffold_test.go`:
```go
package scaffold

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitCreatesFilesAndSkipsExisting(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "homonto.toml"), []byte("# mine\n"), 0o644)

	created, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range created {
		if filepath.Base(p) == "homonto.toml" {
			t.Fatal("must not recreate existing homonto.toml")
		}
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "homonto.toml")); string(b) != "# mine\n" {
		t.Fatal("existing config overwritten")
	}
	if _, err := os.Stat(filepath.Join(dir, ".gitignore")); err != nil {
		t.Fatal(".gitignore not created")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/scaffold/`
Expected: FAIL — `undefined: Init`.

- [ ] **Step 3: Write minimal implementation**

Create `internal/scaffold/scaffold.go`:
```go
package scaffold

import (
	"os"
	"path/filepath"
)

var files = map[string]string{
	"homonto.toml": `# homonto — declarative config for AI coding tools.
# Secrets are referenced, never stored: use ${pass:path} or ${ENV_VAR}.

# [mcps.codegraph]
# command = ["codegraph", "serve", "--mcp"]
# targets = ["claude", "opencode"]   # default: all

# [skills]
# own = ["graphify"]

# [plugins]
# claude = ["claude-hud@official"]
# opencode = ["@slkiser/opencode-quota"]

# [settings.claude]
# model = "opus"
`,
	".gitignore":    "/.homonto/\n.env\n",
	".env.example":  "# Document non-pass secrets here, then copy to .env (gitignored).\n# BRAVE_API_KEY=\n",
}

// Init scaffolds a homonto repo, skipping files that already exist.
func Init(dir string) ([]string, error) {
	var created []string
	for name, body := range files {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			continue
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			return created, err
		}
		created = append(created, p)
	}
	keep := filepath.Join(dir, "content", "skills", ".gitkeep")
	if _, err := os.Stat(keep); err != nil {
		if err := os.MkdirAll(filepath.Dir(keep), 0o755); err != nil {
			return created, err
		}
		if err := os.WriteFile(keep, nil, 0o644); err != nil {
			return created, err
		}
		created = append(created, keep)
	}
	return created, nil
}
```

Create `internal/cli/init.go`:
```go
package cli

import (
	"github.com/noviopenworks/homonto/internal/scaffold"
	"github.com/spf13/cobra"
)

func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init [dir]",
		Short: "Scaffold a new homonto repo",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) == 1 {
				dir = args[0]
			}
			created, err := scaffold.Init(dir)
			if err != nil {
				return err
			}
			for _, p := range created {
				cmd.Println("created", p)
			}
			return nil
		},
	}
}
```

Register in `NewRootCmd`: add `initCmd()` to the `AddCommand` call.

- [ ] **Step 4: Run tests + build**

Run: `go test ./... && go build ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/scaffold/ internal/cli/init.go internal/cli/root.go
git commit -m "feat: init command to scaffold a homonto repo"
```

---

### Task 13: `import` command (bootstrap from existing setup)

**Files:**
- Create: `internal/importer/importer.go`
- Create: `internal/cli/import.go`
- Test: `internal/importer/importer_test.go`

**Interfaces:**
- Consumes: `jsonutil`, `config` types, `pelletier/go-toml/v2` for marshaling.
- Produces:
  - `Import(home string) (*config.Config, error)` — read `home/.claude.json` (mcpServers), `home/.claude/settings.json` (model + enabledPlugins), `home/.config/opencode/opencode.jsonc` (mcp + plugin + model); build a `config.Config`. **Secret heuristic:** any string value that looks like a literal secret (matches `^(sk-|github_pat_|ghp_)` or key name ends in `_KEY`/`_TOKEN` with a non-`${` value) is replaced with `${pass:imported/<name>}` and recorded in a returned `warnings []string` so nothing secret is written.
  - `Import` returns `(*config.Config, []string, error)`.
  - `MarshalTOML(c *config.Config) ([]byte, error)` — serialize to TOML.
  - CLI `homonto import` writes `homonto.toml` (refusing to overwrite unless `--force`) and prints warnings.

- [ ] **Step 1: Write the failing test (mcp import + secret redaction)**

Create `internal/importer/importer_test.go`:
```go
package importer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImportRedactsSecretsInEnv(t *testing.T) {
	home := t.TempDir()
	os.WriteFile(filepath.Join(home, ".claude.json"), []byte(`{
	  "mcpServers": {
	    "brave": {"command":["npx","server-brave"],"env":{"BRAVE_API_KEY":"sk-secret-123"}}
	  }
	}`), 0o644)

	c, warnings, err := Import(home)
	if err != nil {
		t.Fatal(err)
	}
	got := c.MCPs["brave"].Env["BRAVE_API_KEY"]
	if !strings.HasPrefix(got, "${pass:") {
		t.Fatalf("secret not redacted: %q", got)
	}
	if strings.Contains(got, "sk-secret-123") {
		t.Fatal("literal secret leaked into config")
	}
	if len(warnings) == 0 {
		t.Fatal("expected a warning about the redacted secret")
	}

	out, err := MarshalTOML(c)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "sk-secret-123") {
		t.Fatal("literal secret leaked into TOML output")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/importer/`
Expected: FAIL — `undefined: Import`.

- [ ] **Step 3: Write minimal implementation**

Create `internal/importer/importer.go`:
```go
package importer

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/jsonutil"
	"github.com/tidwall/gjson"
)

var secretLike = regexp.MustCompile(`^(sk-|github_pat_|ghp_|xox)`)

// Import reads existing tool config into a homonto Config, redacting secrets.
func Import(home string) (*config.Config, []string, error) {
	c := &config.Config{MCPs: map[string]config.MCP{}}
	var warnings []string

	mj, err := os.ReadFile(filepath.Join(home, ".claude.json"))
	if err == nil {
		doc, _ := jsonutil.Standardize(mj)
		gjson.GetBytes(doc, "mcpServers").ForEach(func(name, server gjson.Result) bool {
			var cmd []string
			for _, v := range server.Get("command").Array() {
				cmd = append(cmd, v.String())
			}
			env := map[string]string{}
			server.Get("env").ForEach(func(k, v gjson.Result) bool {
				val := v.String()
				if redacted, hit := redact(name.String(), k.String(), val); hit {
					warnings = append(warnings, fmt.Sprintf("redacted %s.%s -> %s", name.String(), k.String(), redacted))
					val = redacted
				}
				env[k.String()] = val
				return true
			})
			m := config.MCP{Command: cmd, Targets: []string{"claude"}}
			if len(env) > 0 {
				m.Env = env
			}
			c.MCPs[name.String()] = m
			return true
		})
	}
	return c, warnings, nil
}

func redact(server, key, val string) (string, bool) {
	if strings.HasPrefix(val, "${") {
		return val, false
	}
	if secretLike.MatchString(val) || strings.HasSuffix(key, "_KEY") || strings.HasSuffix(key, "_TOKEN") {
		return fmt.Sprintf("${pass:imported/%s/%s}", server, key), true
	}
	return val, false
}

// MarshalTOML serializes a Config to TOML.
func MarshalTOML(c *config.Config) ([]byte, error) { return toml.Marshal(c) }
```

Create `internal/cli/import.go`:
```go
package cli

import (
	"os"

	"github.com/noviopenworks/homonto/internal/importer"
	"github.com/spf13/cobra"
)

func importCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Bootstrap homonto.toml from your current setup",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfgPath, _ := cmd.Flags().GetString("config")
			force, _ := cmd.Flags().GetBool("force")
			if _, err := os.Stat(cfgPath); err == nil && !force {
				cmd.Printf("%s already exists; use --force to overwrite\n", cfgPath)
				return nil
			}
			home, _ := os.UserHomeDir()
			c, warnings, err := importer.Import(home)
			if err != nil {
				return err
			}
			data, err := importer.MarshalTOML(c)
			if err != nil {
				return err
			}
			if err := os.WriteFile(cfgPath, data, 0o644); err != nil {
				return err
			}
			cmd.Println("wrote", cfgPath)
			for _, w := range warnings {
				cmd.Println("  warn:", w)
			}
			return nil
		},
	}
	cmd.Flags().Bool("force", false, "overwrite existing config")
	return cmd
}
```

Register in `NewRootCmd`: add `importCmd()`.

- [ ] **Step 4: Run all tests + build**

Run: `go test ./... && go build ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/importer/ internal/cli/import.go internal/cli/root.go
git commit -m "feat: import command to bootstrap config from existing setup"
```

---

### Task 14: README + end-to-end smoke test

**Files:**
- Create: `README.md`
- Create: `internal/engine/e2e_test.go`

**Interfaces:**
- Consumes: everything. Proves the full pipeline against a temp `$HOME`.

- [ ] **Step 1: Write the end-to-end test**

Create `internal/engine/e2e_test.go`:
```go
package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/noviopenworks/homonto/internal/secret"
	"github.com/tidwall/gjson"
)

const e2eTOML = `
[mcps.codegraph]
command = ["codegraph","serve","--mcp"]

[skills]
own = ["graphify"]

[settings.claude]
model = "opus"
`

func TestEndToEndApplyIsIdempotent(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	os.WriteFile(filepath.Join(repo, "homonto.toml"), []byte(e2eTOML), 0o644)
	os.MkdirAll(filepath.Join(repo, "content", "skills", "graphify"), 0o755)

	build := func() *Engine {
		e, err := Build(filepath.Join(repo, "homonto.toml"), home, filepath.Join(repo, "content"))
		if err != nil {
			t.Fatal(err)
		}
		e.Resolver = &secret.Resolver{Getenv: os.Getenv, Pass: func(string) (string, error) { return "", nil }}
		return e
	}

	e := build()
	sets, _ := e.Plan()
	if err := e.Apply(sets); err != nil {
		t.Fatal(err)
	}

	// codegraph projected into both tools
	cj, _ := os.ReadFile(filepath.Join(home, ".claude.json"))
	if gjson.GetBytes(cj, "mcpServers.codegraph.command.0").String() != "codegraph" {
		t.Fatal("claude mcp missing")
	}
	oc, _ := os.ReadFile(filepath.Join(home, ".config", "opencode", "opencode.jsonc"))
	if gjson.GetBytes(oc, "mcp.codegraph.type").String() != "local" {
		t.Fatal("opencode mcp missing")
	}
	// skill linked
	if _, err := os.Lstat(filepath.Join(home, ".claude", "skills", "graphify")); err != nil {
		t.Fatal("claude skill link missing")
	}

	// Second apply: no changes.
	e2 := build()
	sets2, _ := e2.Plan()
	if hasReal(sets2) {
		t.Fatalf("second apply not idempotent: %+v", sets2)
	}
}

func hasReal(sets []interface{ }) bool { return false } // replaced below
```

Note: replace the bogus `hasReal` helper — import `plan` and use `plan.HasChanges(sets2)` directly instead of the placeholder. Final assertion line: `if plan.HasChanges(sets2) { t.Fatalf(...) }`, and add `"github.com/noviopenworks/homonto/internal/plan"` to imports; delete the `hasReal` function.

- [ ] **Step 2: Run the e2e test**

Run: `go test ./internal/engine/ -run EndToEnd -v`
Expected: PASS. If the second apply reports changes, the bug is value-formatting mismatch between `desired()` and `current()` in an adapter (normalize both through `json.Marshal(unmarshal(...))` as noted in Task 8 Step 4).

- [ ] **Step 3: Write the README**

Create `README.md` covering: what homonto is, install (`go install github.com/noviopenworks/homonto@latest`), quickstart (`homonto init` → edit `homonto.toml` → `homonto plan` → `homonto apply`), the secret-reference syntax (`${pass:…}`, `${ENV}`), the **JSONC comment caveat** for `opencode.jsonc`, and that owned content is symlinked from `content/`.

- [ ] **Step 4: Run the full suite + build**

Run: `go test ./... && go vet ./... && go build ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add README.md internal/engine/e2e_test.go
git commit -m "test: end-to-end apply + README"
```

---

## Self-Review

**Spec coverage:**
- Declarative source of truth, TOML → Task 2. ✓
- Reference-only secrets (`pass`/env), resolved after confirm, all-at-once → Tasks 3, 10 (two-phase). ✓
- Own-local content via symlinks → Tasks 7, 9. ✓
- Surgical merge, preserve unmanaged keys → Tasks 5, 8, 9 (golden-style assertions). ✓
- Plan/confirm/apply, idempotent, drift → Tasks 6, 10, 11, 14. ✓
- All 4 concepts (MCP, skills, plugins, settings) × both tools → Tasks 8, 9. ✓
- CLI: init/import/plan/apply/status/doctor → Tasks 10–13. ✓
- Error handling: two-phase abort (Task 10), symlink conflict (Task 7), secret never in plan (Task 6), missing skill (Task 11). ✓
- JSONC caveat documented → Task 14 README. ✓
- Stack matches spec (cobra, go-toml/v2, sjson/gjson, hujson). ✓

**Placeholder scan:** The only intentional placeholder is the bogus `hasReal` in Task 14 Step 1, which the step's own note instructs to replace with `plan.HasChanges`. All other steps carry real code.

**Type consistency:** `adapter.Change{Action,Key,Old,New}`, `ChangeSet{Tool,Changes}`, `Adapter{Name,Plan,Apply}`, `secret.Resolver{Getenv,Pass}`, `state.State{Get,Set,Load,Save}`, `Engine{Plan,Apply,Drift,Doctor,ContentDir}` are used consistently across tasks. Adapter constructors converge on `New(home, content string)` after Task 9 (Task 8 introduces `New(home)` then Task 9 Step 1 explicitly migrates it and its test).

## Known follow-ups (post-v1, from spec "Open questions")
- Encrypted in-repo secrets (age/sops); additional resolvers (`op`, keychain).
- More adapters (Codex, Cursor, Gemini) via the `Adapter` interface.
- `--copy` mode; profiles/per-machine overlays.
- Preserve JSONC comments through merge (currently dropped in rewritten regions).
