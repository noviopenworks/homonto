---
change: onto-binary-authoritative-state
design-doc: docs/superpowers/specs/2026-07-13-onto-binary-authoritative-state-design.md
base-ref: cad5274b6d859e66de874edd68bb994c2e97b774
---

# onto-binary-authoritative-state Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the onto Go binary the single authority for onto workflow state — a versioned, typed, shape-validated `onto-state.yaml` that migrates both legacy shapes on read, exposes a CLI command for every gated mutation plus a structured read, and enumerates-then-classifies change directories so a deleted state file surfaces instead of vanishing.

**Architecture:** Extend `internal/ontostate` from a flat 7-field struct to a versioned schema whose gated **core** fields validate for presence/shape (never judgment, B1) and whose **observational** fields are carried but never gate. `Load` migrates legacy inputs on read; a directory-aware `LoadChange` folds a co-resident legacy `state.yaml` and fails loud when their gated cores disagree. `internal/ontocli` gains an `onto set <field>` transition group and `onto state <change> --json`, and inverts `status`/`doctor` to enumerate directories first, then classify each `valid` / `malformed` / `missing-state`.

**Tech Stack:** Go, `gopkg.in/yaml.v3` (only `ontostate` may import it), `encoding/json` (structured read), `github.com/spf13/cobra` (CLI), standard `testing`.

## Global Constraints

- `internal/ontostate` is the ONLY package permitted to import `gopkg.in/yaml.v3` (existing package doc rule) — the CLI never parses YAML itself; it goes through `ontostate`.
- B1 gating: validation checks **presence and shape only** (enum membership / format). A field value outside its allowed set is rejected; a "substantively unconvincing but well-formed" value is NEVER rejected.
- Gated **core** fields (validated): change, workflow (`full|fix|tweak`), phase (`open|design|build|verify|close`), created, base_ref, deps, isolation (`branch|worktree|""`), build_mode (`direct|subagent|""`), tdd_mode (`tdd|direct|""`), verify.scale (`light|full|""`), verify.result (`pending|pass|fail`), close.merged (bool), archived (bool), directive (free string).
- **Observational** fields (carried, NEVER gated): metrics (phase→date map), tasks_total, verify_rounds, preset_escalated.
- Every write emits the current `schema_version` (currently `1`).
- Parsing invalid/malformed state returns a clear error naming the file — it never panics.
- `onto status` and `onto doctor` remain strictly read-only and config-independent (no `homonto.toml`, no engine, zero writes).
- State files live under `docs/changes/<name>/` — canonical filename `onto-state.yaml`; the legacy skill file is `state.yaml`. The single `*` glob under `docs/changes/` excludes `docs/changes/archive/`.
- NON-GOALS — do NOT touch these here: the `onto*` skills / the markdown-only `state.yaml` copy (that is change B `onto-skills-shell-out`); semantic gate *content*, workflow-aware transitions, dep resolver (that is N2); any homonto-engine / projection-pipeline work.
- Verification gate for the whole change: `go test ./internal/ontostate/... ./internal/ontocli/... -race`, `go vet ./...`, `go build ./...`, `openspec validate --all`.

### Grouping decision (delegated by the design)

The design shows `State{ SchemaVersion; Core; Observed }`. The design explicitly delegates exact grouping to this plan ("Exact command names/grouping are refined in the implementation plan"). This plan implements the **gated core as flat top-level fields on `State`** (preserving the legacy binary `onto-state.yaml` wire format — those fields are already top-level — and every existing `st.Phase` / `State{Change: …}` call site) and the **observational group as a nested `Observed` field**. `Validate()` inspects only the core fields and never reads `Observed`, which delivers the design's behavioral contract (an unknown/garbage observational field can never break a gate) without a wire-format break or a wide call-site refactor.

---

## Task 1: Versioned typed schema + shape validation + round-trip

**Files:**
- Modify: `internal/ontostate/state.go` (extend `State`, add `Verify`/`Close`/`Observed` types + `CurrentSchemaVersion`; extend `Validate`; stamp version in `Save`)
- Test: `internal/ontostate/state_test.go`

**Interfaces:**
- Consumes: nothing (foundation).
- Produces:
  - `const CurrentSchemaVersion = 1`
  - `type Verify struct { Scale string; Result string }`
  - `type Close struct { Merged bool }`
  - `type Observed struct { Metrics map[string]string; TasksTotal int; VerifyRounds int; PresetEscalated bool }`
  - `State` gains fields: `SchemaVersion int`, `Isolation string`, `BuildMode string`, `TDDMode string`, `Verify Verify`, `Close Close`, `Directive string`, `Observed Observed` (existing `Change/Workflow/Phase/Created/BaseRef/Deps/Archived` unchanged).
  - `func (s State) Validate() error` — now also enum-checks the optional core fields when non-empty.
  - `func Save(path string, s State) error` — unchanged signature; now stamps `s.SchemaVersion = CurrentSchemaVersion` before marshal.

Note: existing `Change`/`Phase` reads, `st.Phase = next`, `st.Archived = true`, and `State{Change: …, Phase: …}` literals in `new.go` / test helpers all keep compiling because the core stays flat. Do NOT restructure them.

- [x] **Step 1: Write the failing round-trip + validation tests**

Append to `internal/ontostate/state_test.go`:

```go
func fullFixtureState() State {
	return State{
		SchemaVersion: CurrentSchemaVersion,
		Change:        "onto-binary-authoritative-state",
		Workflow:      "full",
		Phase:         "verify",
		Created:       "2026-07-13",
		BaseRef:       "cad5274",
		Deps:          []string{"onto-binary-foundation"},
		Isolation:     "worktree",
		BuildMode:     "subagent",
		TDDMode:       "tdd",
		Verify:        Verify{Scale: "full", Result: "pass"},
		Close:         Close{Merged: true},
		Directive:     "user said: ship it without asking again",
		Archived:      false,
		Observed: Observed{
			Metrics:         map[string]string{"open": "2026-07-13", "build": "2026-07-13"},
			TasksTotal:      9,
			VerifyRounds:    2,
			PresetEscalated: true,
		},
	}
}

func TestMarshalParse_RoundTrip_PreservesEveryGatedField(t *testing.T) {
	want := fullFixtureState()
	b, err := Marshal(want)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	got, err := Parse(b)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("round-trip mismatch:\n got  %+v\n want %+v", got, want)
	}
}

func TestSave_StampsCurrentSchemaVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "onto-state.yaml")
	// caller left SchemaVersion at zero; Save must stamp it.
	if err := Save(path, State{Change: "x", Phase: "open"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.SchemaVersion != CurrentSchemaVersion {
		t.Errorf("SchemaVersion = %d, want %d", got.SchemaVersion, CurrentSchemaVersion)
	}
}

func TestValidate_MalformedEnum_Rejected(t *testing.T) {
	cases := map[string]State{
		"isolation":     {Change: "c", Phase: "open", Isolation: "vm"},
		"build_mode":    {Change: "c", Phase: "open", BuildMode: "manual"},
		"tdd_mode":      {Change: "c", Phase: "open", TDDMode: "maybe"},
		"verify.scale":  {Change: "c", Phase: "open", Verify: Verify{Scale: "medium"}},
		"verify.result": {Change: "c", Phase: "open", Verify: Verify{Result: "green"}},
		"workflow":      {Change: "c", Phase: "open", Workflow: "epic"},
	}
	for name, st := range cases {
		if err := st.Validate(); err == nil {
			t.Errorf("Validate() accepted malformed %s, want error", name)
		}
	}
}

func TestValidate_EmptyOptionalEnums_Accepted(t *testing.T) {
	// legacy-tolerant: an unset optional enum ("") is valid shape.
	st := State{Change: "c", Phase: "open"}
	if err := st.Validate(); err != nil {
		t.Errorf("Validate() rejected empty optionals: %v", err)
	}
}
```

- [x] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/ontostate/ -run 'RoundTrip|StampsCurrentSchemaVersion|MalformedEnum|EmptyOptionalEnums' -v`
Expected: compile error / FAIL — `CurrentSchemaVersion`, `Verify`, `Close`, `Observed`, and the new `State` fields do not exist yet.

- [x] **Step 3: Extend the schema and validation in `state.go`**

Add near the top of `internal/ontostate/state.go` (after `validPhases`):

```go
// CurrentSchemaVersion is the schema_version every write emits.
const CurrentSchemaVersion = 1

// enum membership sets for optional gated core fields. An empty value is
// always allowed (legacy-tolerant); a non-empty value must be a member.
var (
	validWorkflows     = map[string]bool{"full": true, "fix": true, "tweak": true}
	validIsolations    = map[string]bool{"branch": true, "worktree": true}
	validBuildModes    = map[string]bool{"direct": true, "subagent": true}
	validTDDModes      = map[string]bool{"tdd": true, "direct": true}
	validVerifyScales  = map[string]bool{"light": true, "full": true}
	validVerifyResults = map[string]bool{"pending": true, "pass": true, "fail": true}
)
```

Replace the `State` struct (lines 26–35) with the versioned schema and its sub-types:

```go
// Verify holds the gated verify-phase fields.
type Verify struct {
	Scale  string `yaml:"scale,omitempty"`  // light | full | ""
	Result string `yaml:"result,omitempty"` // pending | pass | fail
}

// Close holds the gated close-phase progress fields.
type Close struct {
	Merged bool `yaml:"merged,omitempty"`
}

// Observed carries observational state. It is never gated: no field here may
// block a transition, and unknown values in it can never break a gate.
type Observed struct {
	Metrics         map[string]string `yaml:"metrics,omitempty"` // phase -> YYYY-MM-DD
	TasksTotal      int               `yaml:"tasks_total,omitempty"`
	VerifyRounds    int               `yaml:"verify_rounds,omitempty"`
	PresetEscalated bool              `yaml:"preset_escalated,omitempty"`
}

// State models onto-state.yaml: a schema_version, a flat gated core, and a
// carried observational group. Validation inspects the core only.
type State struct {
	SchemaVersion int `yaml:"schema_version,omitempty"`

	// gated core
	Change    string   `yaml:"change"`
	Workflow  string   `yaml:"workflow,omitempty"`
	Phase     string   `yaml:"phase"`
	Created   string   `yaml:"created,omitempty"`
	BaseRef   string   `yaml:"base_ref,omitempty"`
	Deps      []string `yaml:"deps,omitempty"`
	Isolation string   `yaml:"isolation,omitempty"`
	BuildMode string   `yaml:"build_mode,omitempty"`
	TDDMode   string   `yaml:"tdd_mode,omitempty"`
	Verify    Verify   `yaml:"verify,omitempty"`
	Close     Close    `yaml:"close,omitempty"`
	Directive string   `yaml:"directive,omitempty"`
	Archived  bool     `yaml:"archived,omitempty"`

	// observational (carried, never gated)
	Observed Observed `yaml:"observed,omitempty"`
}
```

Replace `Validate` (lines 70–78) with presence + shape checks:

```go
// Validate checks presence and shape of the gated core: Change and Phase must
// be present/known, and every optional enum, when non-empty, must be a member
// of its set. It never inspects Observed (B1: shape, not judgment).
func (s State) Validate() error {
	if s.Change == "" {
		return fmt.Errorf("onto-state: change is required")
	}
	if !validPhases[s.Phase] {
		return fmt.Errorf("onto-state: phase %q is not one of open|design|build|verify|close", s.Phase)
	}
	if s.Workflow != "" && !validWorkflows[s.Workflow] {
		return fmt.Errorf("onto-state: workflow %q is not one of full|fix|tweak", s.Workflow)
	}
	if s.Isolation != "" && !validIsolations[s.Isolation] {
		return fmt.Errorf("onto-state: isolation %q is not one of branch|worktree", s.Isolation)
	}
	if s.BuildMode != "" && !validBuildModes[s.BuildMode] {
		return fmt.Errorf("onto-state: build_mode %q is not one of direct|subagent", s.BuildMode)
	}
	if s.TDDMode != "" && !validTDDModes[s.TDDMode] {
		return fmt.Errorf("onto-state: tdd_mode %q is not one of tdd|direct", s.TDDMode)
	}
	if s.Verify.Scale != "" && !validVerifyScales[s.Verify.Scale] {
		return fmt.Errorf("onto-state: verify.scale %q is not one of light|full", s.Verify.Scale)
	}
	if s.Verify.Result != "" && !validVerifyResults[s.Verify.Result] {
		return fmt.Errorf("onto-state: verify.result %q is not one of pending|pass|fail", s.Verify.Result)
	}
	return nil
}
```

In `Save` (lines 96–114), stamp the version before marshaling — change the body to set it on a local copy so callers need not:

```go
func Save(path string, s State) error {
	s.SchemaVersion = CurrentSchemaVersion
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("onto-state: failed to create directory for %s: %w", path, err)
	}
	// ... rest of the existing body unchanged (Marshal, temp write, rename) ...
}
```

- [x] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/ontostate/ -run 'RoundTrip|StampsCurrentSchemaVersion|MalformedEnum|EmptyOptionalEnums' -v`
Expected: PASS.

- [x] **Step 5: Run the full ontostate + ontocli suites to confirm no regression**

Run: `go test ./internal/ontostate/... ./internal/ontocli/... && go build ./...`
Expected: PASS / build clean. (The added `schema_version` line in saved files is invisible to the field-based assertions in the existing CLI tests; the flat core keeps every literal compiling.)

- [x] **Step 6: Commit**

```bash
git add internal/ontostate/state.go internal/ontostate/state_test.go
git commit -m "feat(ontostate): versioned typed schema with core/observed split and shape validation"
```

---

## Task 2: On-read migration from both legacy shapes

**Files:**
- Create: `internal/ontostate/migrate.go` (legacy struct + migration + `parseAndMigrate`)
- Modify: `internal/ontostate/state.go` (`Load` routes through `parseAndMigrate`)
- Test: `internal/ontostate/migrate_test.go`

**Interfaces:**
- Consumes: `State`, `Verify`, `Close`, `Observed`, `CurrentSchemaVersion` (Task 1).
- Produces:
  - `func parseAndMigrate(b []byte, sourceName string) (State, error)` — parses raw bytes; if `schema_version` is absent, up-migrates the legacy shape to the current schema (ordered, idempotent); a current-version doc is returned as-is. Stamps `SchemaVersion = CurrentSchemaVersion` on the result.
  - `func isLegacy(b []byte) bool` — true when the bytes carry no `schema_version` key (used by Task 3's conflict policy).
  - `Load(path)` now returns migrated state.

- [x] **Step 1: Write the failing migration tests**

Create `internal/ontostate/migrate_test.go`:

```go
package ontostate

import (
	"reflect"
	"testing"
)

// legacy 7-field binary onto-state.yaml (no schema_version).
const legacyBinaryYAML = `change: legacy-bin
workflow: fix
phase: build
created: "2026-07-01"
base_ref: main
deps:
  - dep-a
archived: false
`

// legacy rich skill state.yaml (no schema_version), mirroring
// catalog/skills/onto/references/state-yaml.md.
const legacyRichYAML = `change: legacy-rich
workflow: full
phase: verify
created: "2026-07-02"
base_ref: abc123
deps: []
decisions:
  isolation: worktree
  execution: subagent
  tdd: tdd
  directive: "ship it"
verify:
  mode: full
  result: pass
close:
  merged: true
guides: pending
metrics:
  phases:
    open: "2026-07-02"
    build: "2026-07-03"
  tasks_total: 4
  verify_rounds: 2
  upgraded: true
archived: false
`

func TestParseAndMigrate_LegacyBinary_ToV1(t *testing.T) {
	got, err := parseAndMigrate([]byte(legacyBinaryYAML), "onto-state.yaml")
	if err != nil {
		t.Fatalf("parseAndMigrate: %v", err)
	}
	if got.SchemaVersion != CurrentSchemaVersion {
		t.Errorf("SchemaVersion = %d, want %d", got.SchemaVersion, CurrentSchemaVersion)
	}
	if got.Change != "legacy-bin" || got.Workflow != "fix" || got.Phase != "build" {
		t.Errorf("core mismatch: %+v", got)
	}
	if !reflect.DeepEqual(got.Deps, []string{"dep-a"}) {
		t.Errorf("Deps = %v, want [dep-a]", got.Deps)
	}
}

func TestParseAndMigrate_LegacyRich_MapsEveryGatedField(t *testing.T) {
	got, err := parseAndMigrate([]byte(legacyRichYAML), "state.yaml")
	if err != nil {
		t.Fatalf("parseAndMigrate: %v", err)
	}
	want := State{
		SchemaVersion: CurrentSchemaVersion,
		Change:        "legacy-rich",
		Workflow:      "full",
		Phase:         "verify",
		Created:       "2026-07-02",
		BaseRef:       "abc123",
		Deps:          []string{},
		Isolation:     "worktree",
		BuildMode:     "subagent", // decisions.execution -> build_mode
		TDDMode:       "tdd",      // decisions.tdd       -> tdd_mode
		Verify:        Verify{Scale: "full", Result: "pass"}, // verify.mode -> scale
		Close:         Close{Merged: true},
		Directive:     "ship it", // decisions.directive -> directive
		Archived:      false,
		Observed: Observed{
			Metrics:         map[string]string{"open": "2026-07-02", "build": "2026-07-03"},
			TasksTotal:      4,
			VerifyRounds:    2,
			PresetEscalated: true, // metrics.upgraded -> preset_escalated
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("rich migration mismatch:\n got  %+v\n want %+v", got, want)
	}
}

func TestParseAndMigrate_CurrentVersion_IsNoOp(t *testing.T) {
	orig := fullFixtureState()
	b, err := Marshal(orig)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	once, err := parseAndMigrate(b, "onto-state.yaml")
	if err != nil {
		t.Fatalf("parseAndMigrate #1: %v", err)
	}
	b2, _ := Marshal(once)
	twice, err := parseAndMigrate(b2, "onto-state.yaml")
	if err != nil {
		t.Fatalf("parseAndMigrate #2: %v", err)
	}
	if !reflect.DeepEqual(once, twice) {
		t.Errorf("migration not idempotent:\n once  %+v\n twice %+v", once, twice)
	}
	if !reflect.DeepEqual(once, orig) {
		t.Errorf("current-version doc changed on migrate:\n got  %+v\n want %+v", once, orig)
	}
}
```

- [x] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/ontostate/ -run 'ParseAndMigrate' -v`
Expected: compile error — `parseAndMigrate` is undefined.

- [x] **Step 3: Implement the migration**

Create `internal/ontostate/migrate.go`:

```go
package ontostate

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// legacyState is a permissive superset that captures BOTH legacy shapes: the
// flat 7-field binary onto-state.yaml and the rich skill state.yaml. Fields
// absent from a given file simply stay zero.
type legacyState struct {
	Change   string   `yaml:"change"`
	Workflow string   `yaml:"workflow"`
	Phase    string   `yaml:"phase"`
	Created  string   `yaml:"created"`
	BaseRef  string   `yaml:"base_ref"`
	Deps     []string `yaml:"deps"`
	Archived bool     `yaml:"archived"`

	Decisions struct {
		Isolation string `yaml:"isolation"`
		Execution string `yaml:"execution"`
		TDD       string `yaml:"tdd"`
		Directive string `yaml:"directive"`
	} `yaml:"decisions"`

	Verify struct {
		Mode   string `yaml:"mode"`
		Result string `yaml:"result"`
	} `yaml:"verify"`

	Close struct {
		Merged bool `yaml:"merged"`
	} `yaml:"close"`

	Metrics struct {
		Phases       map[string]string `yaml:"phases"`
		TasksTotal   int               `yaml:"tasks_total"`
		VerifyRounds int               `yaml:"verify_rounds"`
		Upgraded     bool              `yaml:"upgraded"`
	} `yaml:"metrics"`
}

// isLegacy reports whether b carries no schema_version key.
func isLegacy(b []byte) bool {
	var probe struct {
		SchemaVersion *int `yaml:"schema_version"`
	}
	if err := yaml.Unmarshal(b, &probe); err != nil {
		return false // let parseAndMigrate surface the parse error
	}
	return probe.SchemaVersion == nil
}

// migrateLegacy maps a legacy shape onto the current State. Fields the legacy
// file lacks map to their zero value (empty string / false / 0), which the
// schema treats as "unset". Renames: execution->build_mode, tdd->tdd_mode,
// verify.mode->verify.scale, metrics.upgraded->preset_escalated. The
// observational-only "guides" field is intentionally not carried (change B
// re-derives it; it is never gated).
func migrateLegacy(l legacyState) State {
	return State{
		SchemaVersion: CurrentSchemaVersion,
		Change:        l.Change,
		Workflow:      l.Workflow,
		Phase:         l.Phase,
		Created:       l.Created,
		BaseRef:       l.BaseRef,
		Deps:          l.Deps,
		Isolation:     l.Decisions.Isolation,
		BuildMode:     l.Decisions.Execution,
		TDDMode:       l.Decisions.TDD,
		Verify:        Verify{Scale: l.Verify.Mode, Result: l.Verify.Result},
		Close:         Close{Merged: l.Close.Merged},
		Directive:     l.Decisions.Directive,
		Archived:      l.Archived,
		Observed: Observed{
			Metrics:         l.Metrics.Phases,
			TasksTotal:      l.Metrics.TasksTotal,
			VerifyRounds:    l.Metrics.VerifyRounds,
			PresetEscalated: l.Metrics.Upgraded,
		},
	}
}

// parseAndMigrate decodes raw state bytes, up-migrating a legacy (unversioned)
// shape to the current schema. A current-version doc is decoded as-is. The
// result always carries the current schema_version. sourceName names the file
// in any error. Migration is ordered and idempotent.
func parseAndMigrate(b []byte, sourceName string) (State, error) {
	if isLegacy(b) {
		var l legacyState
		if err := yaml.Unmarshal(b, &l); err != nil {
			return State{}, fmt.Errorf("onto-state: %s: %w", sourceName, err)
		}
		return migrateLegacy(l), nil
	}
	st, err := Parse(b)
	if err != nil {
		return State{}, err
	}
	st.SchemaVersion = CurrentSchemaVersion
	return st, nil
}
```

Route `Load` through it — replace the body of `Load` (lines 55–65 of `state.go`):

```go
func Load(path string) (State, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return State{}, fmt.Errorf("onto-state: failed to read %s: %w", path, err)
	}
	state, err := parseAndMigrate(b, path)
	if err != nil {
		return State{}, fmt.Errorf("%s: %w", path, err)
	}
	return state, nil
}
```

- [x] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/ontostate/ -run 'ParseAndMigrate' -v`
Expected: PASS.

- [x] **Step 5: Run the full ontostate + ontocli suites**

Run: `go test ./internal/ontostate/... ./internal/ontocli/... && go vet ./internal/ontostate/...`
Expected: PASS. (`Load`-based callers — advance/close/status/doctor — now transparently migrate legacy inputs; existing fixtures without `schema_version` still load.)

- [x] **Step 6: Commit**

```bash
git add internal/ontostate/migrate.go internal/ontostate/migrate_test.go internal/ontostate/state.go
git commit -m "feat(ontostate): migrate legacy binary and rich skill state on read"
```

---

## Task 3: Directory-aware LoadChange, both-legacy conflict policy, and Classify

**Files:**
- Modify: `internal/ontostate/migrate.go` (add `LoadChange`, `Classify`, `coreAgrees`, `mergeObserved`)
- Test: `internal/ontostate/loadchange_test.go`

**Interfaces:**
- Consumes: `parseAndMigrate`, `isLegacy`, `State`, `Observed`, `Validate` (Tasks 1–2).
- Produces:
  - `func LoadChange(changeDir string) (State, error)` — resolves `onto-state.yaml` and/or `state.yaml` in `changeDir`. One present → migrate it. Both present and BOTH legacy with disagreeing gated core (phase, workflow, or archived) → malformed error naming the conflict. Otherwise merges Observed (union; the skill file's richer per-field value wins) onto the canonical `onto-state.yaml` state. Neither present → error.
  - `func Classify(changeDir string) (State, string, error)` — returns class `"valid"` / `"malformed"` / `"missing-state"`. Missing both files → `("", "missing-state", nil)`; load or validate error → `("", "malformed", err)`; else `(state, "valid", nil)`.

- [x] **Step 1: Write the failing LoadChange + Classify tests**

Create `internal/ontostate/loadchange_test.go`:

```go
package ontostate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFixture(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func TestLoadChange_BothLegacyAgree_MergesObserved(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "onto-state.yaml", "change: c\nworkflow: full\nphase: build\narchived: false\n")
	writeFixture(t, dir, "state.yaml", "change: c\nworkflow: full\nphase: build\narchived: false\nmetrics:\n  phases:\n    open: \"2026-07-10\"\n  tasks_total: 3\n")

	st, err := LoadChange(dir)
	if err != nil {
		t.Fatalf("LoadChange: %v", err)
	}
	if st.Phase != "build" {
		t.Errorf("Phase = %q, want build", st.Phase)
	}
	if st.Observed.TasksTotal != 3 || st.Observed.Metrics["open"] != "2026-07-10" {
		t.Errorf("Observed not merged from skill file: %+v", st.Observed)
	}
}

func TestLoadChange_BothLegacyDisagree_IsMalformed(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "onto-state.yaml", "change: c\nphase: build\n")
	writeFixture(t, dir, "state.yaml", "change: c\nphase: verify\n") // phase disagrees

	_, err := LoadChange(dir)
	if err == nil {
		t.Fatal("LoadChange accepted disagreeing dual legacy files, want malformed error")
	}
	if !strings.Contains(err.Error(), "phase") {
		t.Errorf("error = %q, want it to name the conflicting field", err.Error())
	}
}

func TestClassify_MissingState(t *testing.T) {
	dir := t.TempDir() // directory exists, no state file
	_, class, err := Classify(dir)
	if err != nil {
		t.Fatalf("Classify: %v", err)
	}
	if class != "missing-state" {
		t.Errorf("class = %q, want missing-state", class)
	}
}

func TestClassify_ValidAndMalformed(t *testing.T) {
	valid := t.TempDir()
	writeFixture(t, valid, "onto-state.yaml", "change: c\nphase: build\n")
	if st, class, err := Classify(valid); err != nil || class != "valid" || st.Phase != "build" {
		t.Errorf("valid case: class=%q phase=%q err=%v", class, st.Phase, err)
	}

	bad := t.TempDir()
	writeFixture(t, bad, "onto-state.yaml", "change: c\nphase: bogus\n")
	if _, class, err := Classify(bad); class != "malformed" || err == nil {
		t.Errorf("malformed case: class=%q err=%v, want malformed + error", class, err)
	}
}
```

- [x] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/ontostate/ -run 'LoadChange|Classify' -v`
Expected: compile error — `LoadChange` and `Classify` are undefined.

- [x] **Step 3: Implement LoadChange, Classify, and helpers**

Append to `internal/ontostate/migrate.go` (add `"os"` and `"path/filepath"` to its imports):

```go
// coreAgrees reports an error if the gated core fields that matter for the
// dual-legacy conflict policy — phase, workflow, archived — disagree.
func coreAgrees(a, b State) error {
	var diffs []string
	if a.Phase != b.Phase {
		diffs = append(diffs, fmt.Sprintf("phase (%q vs %q)", a.Phase, b.Phase))
	}
	if a.Workflow != b.Workflow {
		diffs = append(diffs, fmt.Sprintf("workflow (%q vs %q)", a.Workflow, b.Workflow))
	}
	if a.Archived != b.Archived {
		diffs = append(diffs, fmt.Sprintf("archived (%v vs %v)", a.Archived, b.Archived))
	}
	if len(diffs) > 0 {
		return fmt.Errorf("disagreeing legacy files: %s", strings.Join(diffs, ", "))
	}
	return nil
}

// mergeObserved unions two Observed groups; the skill file's richer per-field
// value wins when set.
func mergeObserved(base, skill Observed) Observed {
	out := base
	if len(skill.Metrics) > 0 {
		if out.Metrics == nil {
			out.Metrics = map[string]string{}
		}
		for k, v := range skill.Metrics {
			out.Metrics[k] = v
		}
	}
	if skill.TasksTotal != 0 {
		out.TasksTotal = skill.TasksTotal
	}
	if skill.VerifyRounds != 0 {
		out.VerifyRounds = skill.VerifyRounds
	}
	if skill.PresetEscalated {
		out.PresetEscalated = true
	}
	return out
}

// LoadChange loads a change's state from changeDir, resolving the canonical
// onto-state.yaml and a co-resident legacy state.yaml. If both are present and
// both legacy, disagreeing gated core (phase/workflow/archived) is malformed;
// otherwise Observed is merged onto the canonical state.
func LoadChange(changeDir string) (State, error) {
	ontoPath := filepath.Join(changeDir, "onto-state.yaml")
	skillPath := filepath.Join(changeDir, "state.yaml")

	ontoB, ontoErr := os.ReadFile(ontoPath)
	skillB, skillErr := os.ReadFile(skillPath)

	switch {
	case ontoErr == nil && skillErr == nil:
		ontoState, err := parseAndMigrate(ontoB, ontoPath)
		if err != nil {
			return State{}, err
		}
		skillState, err := parseAndMigrate(skillB, skillPath)
		if err != nil {
			return State{}, err
		}
		if isLegacy(ontoB) && isLegacy(skillB) {
			if err := coreAgrees(ontoState, skillState); err != nil {
				return State{}, fmt.Errorf("%s: %w", changeDir, err)
			}
		}
		ontoState.Observed = mergeObserved(ontoState.Observed, skillState.Observed)
		return ontoState, nil
	case ontoErr == nil:
		return parseAndMigrate(ontoB, ontoPath)
	case skillErr == nil:
		return parseAndMigrate(skillB, skillPath)
	default:
		return State{}, fmt.Errorf("onto-state: no state file (onto-state.yaml or state.yaml) in %s", changeDir)
	}
}

// Classify loads and validates a change directory, returning one of
// "valid", "malformed", or "missing-state". A directory with no state file at
// all is "missing-state" (not an error) so a deleted state file is reported
// rather than silently skipped (F14).
func Classify(changeDir string) (State, string, error) {
	ontoMissing := fileMissing(filepath.Join(changeDir, "onto-state.yaml"))
	skillMissing := fileMissing(filepath.Join(changeDir, "state.yaml"))
	if ontoMissing && skillMissing {
		return State{}, "missing-state", nil
	}
	st, err := LoadChange(changeDir)
	if err != nil {
		return State{}, "malformed", err
	}
	if err := st.Validate(); err != nil {
		return State{}, "malformed", err
	}
	return st, "valid", nil
}

func fileMissing(path string) bool {
	_, err := os.Stat(path)
	return os.IsNotExist(err)
}
```

- [x] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/ontostate/ -run 'LoadChange|Classify' -v`
Expected: PASS.

- [x] **Step 5: Run the full ontostate suite under -race**

Run: `go test ./internal/ontostate/... -race`
Expected: PASS.

- [x] **Step 6: Commit**

```bash
git add internal/ontostate/migrate.go internal/ontostate/loadchange_test.go
git commit -m "feat(ontostate): directory-aware LoadChange with dual-legacy conflict policy and Classify"
```

---

## Task 4: `onto set` transition group — enum setters

**Files:**
- Create: `internal/ontocli/set.go` (`set` parent command, transition helper, enum setter subcommands)
- Modify: `internal/ontocli/root.go` (register `setCmd()`)
- Test: `internal/ontocli/set_test.go`

**Interfaces:**
- Consumes: `gate` (`init.go`), `validChangeName` (`new.go`), `ontostate.LoadChange`, `ontostate.Save`, `ontostate.State`, `(State).Validate` (Tasks 1–3).
- Produces:
  - `func setCmd() *cobra.Command` — the `onto set <field> <change> <value>` parent, with subcommands `isolation`, `build-mode`, `tdd-mode`, `verify-scale`, `verify-result`.
  - `func runTransition(cmd *cobra.Command, root, name string, apply func(*ontostate.State) error) error` — loads via `LoadChange`, applies `apply` (which validates+mutates), re-runs `Validate`, then `Save`. Gated on `gate(root)` and `validChangeName(name)`. Writes nothing on any failure.

Each setter is semantic-per-field (its own subcommand owns its allowed set) — NOT a generic `set <key> <value>`.

- [x] **Step 1: Write the failing enum-setter tests**

Create `internal/ontocli/set_test.go`. (Reuses `prepWorkspace`, `seedChange`, `writeFile` from the existing package test helpers.)

```go
package ontocli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/noviopenworks/homonto/internal/ontostate"
)

func runOnto(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestSetIsolation_HappyPath_WritesField(t *testing.T) {
	root := prepWorkspace(t)
	seedChange(t, root, "c", "build")

	if _, err := runOnto(t, "set", "isolation", "c", "worktree", "--dir", root); err != nil {
		t.Fatalf("set isolation: %v", err)
	}
	st, err := ontostate.LoadChange(filepath.Join(root, "docs", "changes", "c"))
	if err != nil {
		t.Fatalf("LoadChange: %v", err)
	}
	if st.Isolation != "worktree" {
		t.Errorf("Isolation = %q, want worktree", st.Isolation)
	}
}

func TestSetIsolation_BadValue_RejectedNoWrite(t *testing.T) {
	root := prepWorkspace(t)
	seedChange(t, root, "c", "build")

	out, err := runOnto(t, "set", "isolation", "c", "vm", "--dir", root)
	if err == nil {
		t.Fatal("set isolation vm succeeded, want rejection")
	}
	if !strings.Contains(out+err.Error(), "isolation") {
		t.Errorf("error = %q, want it to name the field", err)
	}
	st, _ := ontostate.LoadChange(filepath.Join(root, "docs", "changes", "c"))
	if st.Isolation != "" {
		t.Errorf("Isolation = %q, want unchanged empty after rejected write", st.Isolation)
	}
}

func TestSetEnumSetters_HappyPaths(t *testing.T) {
	root := prepWorkspace(t)
	seedChange(t, root, "c", "build")

	cases := []struct {
		field, value string
		read         func(ontostate.State) string
	}{
		{"build-mode", "subagent", func(s ontostate.State) string { return s.BuildMode }},
		{"tdd-mode", "tdd", func(s ontostate.State) string { return s.TDDMode }},
		{"verify-scale", "full", func(s ontostate.State) string { return s.Verify.Scale }},
		{"verify-result", "pass", func(s ontostate.State) string { return s.Verify.Result }},
	}
	for _, tc := range cases {
		if _, err := runOnto(t, "set", tc.field, "c", tc.value, "--dir", root); err != nil {
			t.Fatalf("set %s: %v", tc.field, err)
		}
		st, _ := ontostate.LoadChange(filepath.Join(root, "docs", "changes", "c"))
		if got := tc.read(st); got != tc.value {
			t.Errorf("after set %s: got %q, want %q", tc.field, got, tc.value)
		}
	}
}
```

- [x] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/ontocli/ -run 'TestSet' -v`
Expected: FAIL — `unknown command "set"`.

- [x] **Step 3: Implement the set group and enum setters**

Create `internal/ontocli/set.go`:

```go
package ontocli

import (
	"fmt"
	"path/filepath"

	"github.com/noviopenworks/homonto/internal/ontostate"
	"github.com/spf13/cobra"
)

// runTransition loads the change via LoadChange (so migration + dual-legacy
// conflict detection apply), lets apply validate+mutate the state, re-validates
// the whole state, and saves. It gates on gate(root) and validChangeName, and
// writes nothing if any step fails.
func runTransition(cmd *cobra.Command, root, name string, apply func(*ontostate.State) error) error {
	if err := gate(root); err != nil {
		return err
	}
	if err := validChangeName(name); err != nil {
		return err
	}
	changeDir := filepath.Join(root, "docs", "changes", name)
	st, err := ontostate.LoadChange(changeDir)
	if err != nil {
		return fmt.Errorf("onto set: loading %s: %w", changeDir, err)
	}
	if err := apply(&st); err != nil {
		return err
	}
	if err := st.Validate(); err != nil {
		return err
	}
	if err := ontostate.Save(filepath.Join(changeDir, "onto-state.yaml"), st); err != nil {
		return fmt.Errorf("onto set: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%s: updated\n", name)
	return nil
}

// enumSetterCmd builds a `set <field> <change> <value>` subcommand that
// accepts only members of allowed and applies set() to the loaded state.
func enumSetterCmd(field string, allowed []string, set func(*ontostate.State, string)) *cobra.Command {
	var dir string
	allowedSet := map[string]bool{}
	for _, v := range allowed {
		allowedSet[v] = true
	}
	cmd := &cobra.Command{
		Use:   field + " <change> <value>",
		Short: "Set the " + field + " field of a change",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, value := args[0], args[1]
			return runTransition(cmd, dir, name, func(st *ontostate.State) error {
				if !allowedSet[value] {
					return fmt.Errorf("onto set %s: %q is not one of %v", field, value, allowed)
				}
				set(st, value)
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&dir, "dir", ".", "workspace root containing the change")
	return cmd
}

// setCmd builds the "onto set" parent with one semantic subcommand per gated
// field. Each subcommand owns its field's allowed set.
func setCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set a gated state field of an active change",
	}
	cmd.AddCommand(enumSetterCmd("isolation", []string{"branch", "worktree"},
		func(s *ontostate.State, v string) { s.Isolation = v }))
	cmd.AddCommand(enumSetterCmd("build-mode", []string{"direct", "subagent"},
		func(s *ontostate.State, v string) { s.BuildMode = v }))
	cmd.AddCommand(enumSetterCmd("tdd-mode", []string{"tdd", "direct"},
		func(s *ontostate.State, v string) { s.TDDMode = v }))
	cmd.AddCommand(enumSetterCmd("verify-scale", []string{"light", "full"},
		func(s *ontostate.State, v string) { s.Verify.Scale = v }))
	cmd.AddCommand(enumSetterCmd("verify-result", []string{"pending", "pass", "fail"},
		func(s *ontostate.State, v string) { s.Verify.Result = v }))
	return cmd
}
```

Register in `internal/ontocli/root.go` — add after `root.AddCommand(doctorCmd())`:

```go
	root.AddCommand(setCmd())
```

- [x] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/ontocli/ -run 'TestSet' -v`
Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add internal/ontocli/set.go internal/ontocli/root.go internal/ontocli/set_test.go
git commit -m "feat(ontocli): add onto set transition group with enum-gated field setters"
```

---

## Task 5: `onto set close-merged` and `onto set directive`

**Files:**
- Modify: `internal/ontocli/set.go` (register two more subcommands)
- Test: `internal/ontocli/set_test.go` (append)

**Interfaces:**
- Consumes: `runTransition`, `setCmd` (Task 4).
- Produces: `onto set close-merged <change>` (no value arg — sets `close.merged=true`, idempotent) and `onto set directive <change> <text>` (free string, presence-only — any non-empty text accepted, empty rejected).

- [x] **Step 1: Write the failing tests**

Append to `internal/ontocli/set_test.go`:

```go
func TestSetCloseMerged_SetsTrueIdempotently(t *testing.T) {
	root := prepWorkspace(t)
	seedChange(t, root, "c", "close")

	for i := 0; i < 2; i++ { // idempotent: running twice is fine
		if _, err := runOnto(t, "set", "close-merged", "c", "--dir", root); err != nil {
			t.Fatalf("set close-merged (run %d): %v", i, err)
		}
	}
	st, _ := ontostate.LoadChange(filepath.Join(root, "docs", "changes", "c"))
	if !st.Close.Merged {
		t.Errorf("Close.Merged = false, want true")
	}
}

func TestSetDirective_StoresVerbatim(t *testing.T) {
	root := prepWorkspace(t)
	seedChange(t, root, "c", "build")

	const text = "ship without re-asking the isolation gate"
	if _, err := runOnto(t, "set", "directive", "c", text, "--dir", root); err != nil {
		t.Fatalf("set directive: %v", err)
	}
	st, _ := ontostate.LoadChange(filepath.Join(root, "docs", "changes", "c"))
	if st.Directive != text {
		t.Errorf("Directive = %q, want %q", st.Directive, text)
	}
}

func TestSetDirective_EmptyRejected(t *testing.T) {
	root := prepWorkspace(t)
	seedChange(t, root, "c", "build")

	if _, err := runOnto(t, "set", "directive", "c", "", "--dir", root); err == nil {
		t.Fatal("empty directive accepted, want rejection")
	}
}
```

- [x] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/ontocli/ -run 'CloseMerged|Directive' -v`
Expected: FAIL — `unknown command "close-merged"` / `"directive"`.

- [x] **Step 3: Implement the two subcommands**

Add these builders to `internal/ontocli/set.go` and register them in `setCmd()`:

```go
// closeMergedCmd sets close.merged=true. It takes no value and is idempotent.
func closeMergedCmd() *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:   "close-merged <change>",
		Short: "Mark a change's close.merged flag true (idempotent)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTransition(cmd, dir, args[0], func(st *ontostate.State) error {
				st.Close.Merged = true
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&dir, "dir", ".", "workspace root containing the change")
	return cmd
}

// directiveCmd stores a free-string directive verbatim; presence-only shape
// (empty rejected — a directive is a real pre-authorization, not a clear).
func directiveCmd() *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:   "directive <change> <text>",
		Short: "Record a verbatim pre-authorization directive on a change",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, text := args[0], args[1]
			return runTransition(cmd, dir, name, func(st *ontostate.State) error {
				if text == "" {
					return fmt.Errorf("onto set directive: text must not be empty")
				}
				st.Directive = text
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&dir, "dir", ".", "workspace root containing the change")
	return cmd
}
```

In `setCmd()`, add before `return cmd`:

```go
	cmd.AddCommand(closeMergedCmd())
	cmd.AddCommand(directiveCmd())
```

- [x] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/ontocli/ -run 'CloseMerged|Directive' -v`
Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add internal/ontocli/set.go internal/ontocli/set_test.go
git commit -m "feat(ontocli): add onto set close-merged and directive transitions"
```

---

## Task 6: `onto state <change> --json` structured read

**Files:**
- Create: `internal/ontocli/statecmd.go`
- Modify: `internal/ontocli/root.go` (register `stateCmd()`)
- Test: `internal/ontocli/statecmd_test.go`

**Interfaces:**
- Consumes: `validChangeName`, `ontostate.LoadChange`, `ontostate.State`, `(State).DerivePhase`.
- Produces: `onto state <change> [--json]` — a read-only command (no `gate`, writes nothing) that emits the full validated state plus derived phase. `--json` emits JSON via `encoding/json`; the read must not mutate the tree.

- [x] **Step 1: Write the failing test**

Create `internal/ontocli/statecmd_test.go`:

```go
package ontocli

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestStateJSON_EmitsFullStateAndDerivedPhase(t *testing.T) {
	root := t.TempDir() // read command needs no framework install
	writeFile(t, filepath.Join(root, "docs", "changes", "c", "onto-state.yaml"),
		"change: c\nworkflow: full\nphase: build\nisolation: worktree\n")

	before := treeSnapshot(t, root)

	out, err := runOnto(t, "state", "c", "--json", "--dir", root)
	if err != nil {
		t.Fatalf("state --json: %v", err)
	}

	var got struct {
		Change       string `json:"change"`
		Phase        string `json:"phase"`
		Isolation    string `json:"isolation"`
		DerivedPhase string `json:"derived_phase"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out)
	}
	if got.Change != "c" || got.Isolation != "worktree" {
		t.Errorf("state = %+v, want change=c isolation=worktree", got)
	}
	if got.DerivedPhase != "build" {
		t.Errorf("derived_phase = %q, want build", got.DerivedPhase)
	}

	after := treeSnapshot(t, root)
	if len(before) != len(after) {
		t.Errorf("state --json mutated the tree: before=%d after=%d files", len(before), len(after))
	}
}
```

- [x] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/ontocli/ -run 'TestStateJSON' -v`
Expected: FAIL — `unknown command "state"`.

- [x] **Step 3: Implement the structured read**

Create `internal/ontocli/statecmd.go`:

```go
package ontocli

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/noviopenworks/homonto/internal/ontostate"
	"github.com/spf13/cobra"
)

// stateCmd builds "onto state <change> [--json]": a read-only structured read
// of a change's full validated state and derived phase. It writes nothing and
// is not gated on the framework install.
func stateCmd() *cobra.Command {
	var (
		dir     string
		asJSON  bool
	)
	cmd := &cobra.Command{
		Use:   "state <change>",
		Short: "Print a change's full state (use --json for a machine-readable read)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if err := validChangeName(name); err != nil {
				return err
			}
			changeDir := filepath.Join(dir, "docs", "changes", name)
			st, err := ontostate.LoadChange(changeDir)
			if err != nil {
				return err
			}
			phase, err := st.DerivePhase()
			if err != nil {
				return err
			}
			if asJSON {
				payload := struct {
					ontostate.State
					DerivedPhase string `json:"derived_phase"`
				}{State: st, DerivedPhase: phase}
				b, err := json.MarshalIndent(payload, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(b))
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s: %s\n", name, phase)
			return nil
		},
	}
	cmd.Flags().StringVar(&dir, "dir", ".", "workspace root containing the change")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit the full state as JSON")
	return cmd
}
```

Add JSON tags to `State` and its sub-types so the `--json` output has stable snake_case keys. In `internal/ontostate/state.go`, extend each field tag, e.g. `Change string \`yaml:"change" json:"change"\``, `Phase string \`yaml:"phase" json:"phase"\``, `Isolation string \`yaml:"isolation,omitempty" json:"isolation,omitempty"\``, and likewise `json:"verify,omitempty"`, `json:"scale,omitempty"`, `json:"result,omitempty"`, `json:"close,omitempty"`, `json:"merged,omitempty"`, `json:"directive,omitempty"`, `json:"observed,omitempty"`, `json:"schema_version,omitempty"`, etc. (Mirror the yaml key on every field.)

Register in `internal/ontocli/root.go` — add after `root.AddCommand(setCmd())`:

```go
	root.AddCommand(stateCmd())
```

- [x] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/ontocli/ -run 'TestStateJSON' -v`
Expected: PASS.

- [x] **Step 5: Run the ontostate suite (JSON tags added) to confirm no regression**

Run: `go test ./internal/ontostate/... ./internal/ontocli/...`
Expected: PASS.

- [x] **Step 6: Commit**

```bash
git add internal/ontocli/statecmd.go internal/ontocli/statecmd_test.go internal/ontocli/root.go internal/ontostate/state.go
git commit -m "feat(ontocli): add onto state --json structured read"
```

---

## Task 7: `onto status` — enumerate directories then classify

**Files:**
- Modify: `internal/ontocli/status.go` (rewrite `runStatus`)
- Test: `internal/ontocli/status_test.go` (update existing assertions + add F14 case)

**Interfaces:**
- Consumes: `ontostate.Classify`, `ontostate.ValidateSkeleton`.
- Produces: `runStatus` enumerates change **directories** under `docs/changes/` (excluding `archive/`) FIRST, then prints one line per directory: `valid` (with derived phase + skeleton note), `malformed` (with the problem), or `missing-state`. Zero writes.

Backward-compat note: the existing `TestStatusCommand_ReportsValidAndInvalidChanges` asserts the malformed line contains `"invalid"`. This task changes the vocabulary to `"malformed"` per the spec; update that assertion in the same task.

- [x] **Step 1: Update the existing test and add the F14 missing-state case**

In `internal/ontocli/status_test.go`, change the malformed assertion in `TestStatusCommand_ReportsValidAndInvalidChanges` from `"invalid"` to `"malformed"`:

```go
	if !strings.Contains(got, "beta:") || !strings.Contains(got, "malformed") {
		t.Errorf("output = %q, want a line for beta containing %q", got, "malformed")
	}
```

Append the deleted-state regression test:

```go
func TestStatusCommand_DeletedStateFile_IsMissingStateRow(t *testing.T) {
	dir := t.TempDir()
	// a change directory that exists but has no state file (deleted)
	if err := os.MkdirAll(filepath.Join(dir, "docs", "changes", "gamma"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	out, err := runOnto(t, "status", "--dir", dir)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if !strings.Contains(out, "gamma") || !strings.Contains(out, "missing-state") {
		t.Errorf("output = %q, want a gamma missing-state row (not silently dropped)", out)
	}
}
```

- [x] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/ontocli/ -run 'TestStatusCommand' -v`
Expected: FAIL — the current glob-based `runStatus` skips the state-less `gamma` directory entirely and still prints `invalid` for `beta`.

- [x] **Step 3: Rewrite `runStatus` to enumerate then classify**

Replace the body of `runStatus` in `internal/ontocli/status.go`:

```go
func runStatus(cmd *cobra.Command, root string) error {
	changesDir := filepath.Join(root, "docs", "changes")
	entries, err := os.ReadDir(changesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // no changes dir: nothing to report, still read-only
		}
		return err
	}

	for _, e := range entries {
		if !e.IsDir() || e.Name() == "archive" {
			continue
		}
		changeDir := filepath.Join(changesDir, e.Name())
		st, class, classErr := ontostate.Classify(changeDir)
		switch class {
		case "missing-state":
			cmd.Printf("%s: missing-state\n", e.Name())
		case "malformed":
			cmd.Printf("%s: malformed (%v)\n", e.Name(), classErr)
		default: // valid
			if skeletonErr := ontostate.ValidateSkeleton(changeDir); skeletonErr != nil {
				cmd.Printf("%s: %s — skeleton: %v\n", st.Change, st.Phase, skeletonErr)
			} else {
				cmd.Printf("%s: %s — skeleton ok\n", st.Change, st.Phase)
			}
		}
	}
	return nil
}
```

Update `status.go` imports: add `"os"`, keep `"path/filepath"` and the `ontostate` import; the `filepath.Glob` call is gone.

- [x] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/ontocli/ -run 'TestStatusCommand' -v`
Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add internal/ontocli/status.go internal/ontocli/status_test.go
git commit -m "feat(ontocli): onto status enumerates change dirs then classifies valid/malformed/missing-state"
```

---

## Task 8: `onto doctor` — missing-state directory is a finding

**Files:**
- Modify: `internal/ontocli/doctor.go` (active-changes section)
- Test: `internal/ontocli/doctor_test.go` (add missing-state finding case)

**Interfaces:**
- Consumes: `ontostate.Classify`, `ontostate.ValidateSkeleton`, `ontostate.DepsResolved`.
- Produces: `runDoctor`'s active-changes section enumerates change **directories** FIRST, then classifies each; `malformed` and `missing-state` are findings (exit non-zero). `valid` changes still get the phase/artifact, deps, and archived checks. Docs-layout and archive-layout sections unchanged.

- [x] **Step 1: Write the failing missing-state finding test**

Append to `internal/ontocli/doctor_test.go` (mirror the file's existing setup style):

```go
func TestDoctor_MissingStateDir_IsFinding(t *testing.T) {
	dir := t.TempDir()
	// full docs layout so only the active-change check fires
	for _, d := range []string{"changes", "specs", "adr", "guides"} {
		if err := os.MkdirAll(filepath.Join(dir, "docs", d), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}
	// a change directory with no state file (deleted)
	if err := os.MkdirAll(filepath.Join(dir, "docs", "changes", "gamma"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	out, err := runOnto(t, "doctor", "--dir", dir)
	if err == nil {
		t.Fatal("doctor exited 0 with a missing-state change dir, want non-zero")
	}
	if !strings.Contains(out, "gamma") || !strings.Contains(out, "missing-state") {
		t.Errorf("output = %q, want a gamma missing-state finding", out)
	}
}
```

(Add `"os"`, `"path/filepath"`, `"strings"` to `doctor_test.go` imports if not already present.)

- [x] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/ontocli/ -run 'TestDoctor_MissingStateDir' -v`
Expected: FAIL — the current glob skips the state-less `gamma` directory, so doctor prints `healthy` and exits 0.

- [x] **Step 3: Rewrite the active-changes section of `runDoctor`**

Replace section 2 (the `active, _ := filepath.Glob(...)` loop) in `internal/ontocli/doctor.go` with directory enumeration + classify:

```go
	// 2. active changes: enumerate change directories first (excluding
	// archive/), then classify. A missing-state or malformed directory is a
	// finding — a deleted state file is reported, never silently skipped.
	changesDir := filepath.Join(root, "docs", "changes")
	if entries, readErr := os.ReadDir(changesDir); readErr == nil {
		for _, e := range entries {
			if !e.IsDir() || e.Name() == "archive" {
				continue
			}
			name := e.Name()
			changeDir := filepath.Join(changesDir, name)
			st, class, classErr := ontostate.Classify(changeDir)
			switch class {
			case "missing-state":
				findings = append(findings, name+": missing-state (change directory has no state file)")
				continue
			case "malformed":
				findings = append(findings, fmt.Sprintf("%s: malformed state: %v", name, classErr))
				continue
			}
			phase := st.Phase
			if skErr := ontostate.ValidateSkeleton(changeDir); skErr != nil {
				findings = append(findings, fmt.Sprintf("%s: phase %s missing artifact: %v", name, phase, skErr))
			}
			if unresolved := ontostate.DepsResolved(root, st.Deps); len(unresolved) > 0 {
				findings = append(findings, fmt.Sprintf("%s: unresolved dependencies: %v", name, unresolved))
			}
			if st.Archived {
				findings = append(findings, name+": active change marked archived: true (belongs under docs/changes/archive/)")
			}
		}
	}
```

Ensure `doctor.go` imports include `"os"`, `"fmt"`, `"path/filepath"`, and `ontostate` (all already present).

- [x] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/ontocli/ -run 'TestDoctor' -v`
Expected: PASS (new case passes; existing doctor tests — healthy workspace, invalid state, deps, archive — still pass because `valid`-path behavior is unchanged and a well-formed change still classifies `valid`).

- [x] **Step 5: Commit**

```bash
git add internal/ontocli/doctor.go internal/ontocli/doctor_test.go
git commit -m "feat(ontocli): onto doctor reports missing-state change dirs as findings"
```

---

## Task 9: Full gate + change-B handoff note

**Files:**
- Modify: `openspec/changes/onto-binary-authoritative-state/tasks.md` (check the outline boxes)
- No source changes — this task is the whole-change verification and the concrete-surface record item 6 in `tasks.md` calls for.

**Interfaces:**
- Consumes: everything from Tasks 1–8.
- Produces: a passing full gate and a recorded final schema + CLI surface for change B (`onto-skills-shell-out`).

- [x] **Step 1: Run the full verification gate**

Run:
```bash
go test ./internal/ontostate/... ./internal/ontocli/... -race
go vet ./...
go build ./...
openspec validate --all
```
Expected: all PASS. If `go vet` flags the embedded-struct field name `ontostate.State` in `statecmd.go`'s anonymous JSON payload, confirm the JSON still marshals (the `State` fields carry `json` tags from Task 6); no vet error is expected.

- [x] **Step 2: Record the concrete schema + CLI surface for change B**

Confirm the shipped surface, so `onto-skills-shell-out` can be authored against concrete commands (do NOT author change B here — NON-GOAL):
- State file: `docs/changes/<name>/onto-state.yaml`, `schema_version: 1`, gated core fields + nested `verify:`/`close:`/`observed:` as implemented in Task 1.
- Commands: `onto set isolation|build-mode|tdd-mode|verify-scale|verify-result <change> <value>`, `onto set close-merged <change>`, `onto set directive <change> <text>`, `onto state <change> --json`.
- Reads/classification: `onto status`, `onto doctor` classify `valid|malformed|missing-state`.

- [x] **Step 3: Check the outline boxes in the change's tasks.md**

Tick the completed items in `openspec/changes/onto-binary-authoritative-state/tasks.md` sections 1–5 (and item 6, the change-B surface record), reflecting what shipped.

- [x] **Step 4: Commit**

```bash
git add openspec/changes/onto-binary-authoritative-state/tasks.md
git commit -m "chore(onto): record onto-binary-authoritative-state completion and change-B surface"
```

---

## Self-Review

**Spec coverage (delta `specs/onto-binary/spec.md`):**
- MODIFIED "onto-state.yaml change-state model" — typed core + observational + `schema_version` + shape validation → Task 1; on-read migration (both legacy shapes, ordered, idempotent) → Task 2; disagreeing dual-legacy → malformed → Task 3; clear error, no panic → preserved (`Parse`'s recover) + Task 2/3 error wrapping.
- MODIFIED "onto status is read-only and config-independent" — enumerate dirs → classify, deleted state → `missing-state` row → Task 7.
- MODIFIED "onto doctor reports workflow and project health" — enumerate → classify, missing-state finding → Task 8 (docs-layout / deps / archive sections preserved).
- ADDED "onto exposes state transitions and a structured read" — enum setters → Task 4; close-merged + directive → Task 5; `onto state --json` → Task 6.
- Testing strategy (round-trip / migration both shapes / conflict / classify+deleted / per-command happy+reject / structured read) → Tasks 1–8 each carry the named test.

**Design four-units mapping:** unit 1 → Task 1; unit 2 → Tasks 2–3; unit 3 → Tasks 4–6; unit 4 → Tasks 7–8.

**Backward-compat / test-update notes surfaced:** the current spec's "no migration" is reversed (Task 2 makes `Load` migrate); `status`'s malformed vocabulary changes `invalid`→`malformed` (Task 7 updates the existing assertion); flat-core grouping keeps every `st.Phase` read and `State{Change: …}` literal in existing `new.go`/test helpers compiling unchanged.

**Placeholder scan:** no TBD/TODO; every code step shows complete code; every test step shows the full test.

**Type consistency:** `CurrentSchemaVersion`, `Verify{Scale,Result}`, `Close{Merged}`, `Observed{Metrics,TasksTotal,VerifyRounds,PresetEscalated}`, `LoadChange`, `Classify`, `parseAndMigrate`, `runTransition`, `setCmd`, `stateCmd` used consistently across tasks.

**Risk flagged:** migration field-map data loss (design's top risk) — mitigated by the Task 2 rich-fixture assertion that every gated field maps before any write path relies on it; the observational-only `guides` field is intentionally not carried (change B re-derives it) and this is called out in `migrateLegacy`.
