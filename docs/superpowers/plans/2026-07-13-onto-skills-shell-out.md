---
change: onto-skills-shell-out
design-doc: docs/superpowers/specs/2026-07-13-onto-skills-shell-out-design.md
base-ref: eef55ce8122fbd85bb66627f64086cacb44ef827
---

# onto-skills-shell-out Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the eight `onto*` markdown skills invoke the `onto` binary for every state mutation (zero direct state-file writes), delete the "markdown-only / no external CLI" claim, and add the three binary commands that close the remaining CLI-surface gaps.

**Architecture:** Two layers. Layer A extends the Go binary additively — a `--workflow` flag on `onto new`, `onto set base-ref` / `onto set deps` / `onto set guides` setters, and a `Guides` gated field on `ontostate.State`. Layer B rewrites each skill's imperative state-write instruction to the mapped `onto` command per the Design Doc field→command table, and drops the observational metric-writing instructions. Layer C enforces the invariant with a scoped grep gate wired into `scripts/gate.sh`.

**Tech Stack:** Go 1.26 (`internal/ontostate`, `internal/ontocli`, cobra CLI, `gopkg.in/yaml.v3`), Markdown skills under `catalog/skills/`, Bash gate scripts under `scripts/`.

## Global Constraints

- **NO `schema_version` bump.** `Guides` is added with `omitempty` and empty is always valid (legacy-tolerant); `ontostate.CurrentSchemaVersion` stays `1`.
- **TDD for all Go tasks (A1–A3).** Failing test first, watch it fail, minimal implementation, watch it pass, commit. One commit per task.
- **Skills are NOT TDD** (B1/B2 are Markdown). They are verified by the grep gate (C1) and `openspec validate --all` (C2).
- **One reviewable commit per task.** Never batch tasks.
- **Reuse existing helpers verbatim:** `runTransition(cmd, root, name, apply func(*ontostate.State) error) error` and `enumSetterCmd(field string, allowed []string, set func(*ontostate.State, string))` in `internal/ontocli/set.go`; `ontostate.Save`, `ontostate.LoadChange`, `ontostate.CurrentSchemaVersion` in `internal/ontostate`.
- **Field-name mapping (skill vocabulary → binary vocabulary), used throughout Layer B:** skill `decisions.execution` → binary `build-mode`; skill `decisions.tdd` → binary `tdd-mode`; skill `verify.mode` → binary `verify-scale`. These are pre-existing renames (see `internal/ontostate/migrate.go` lines 59–63); do not introduce new aliases.
- **NON-GOALS — do not implement:** workflow-aware transition *rules*, semantic gates, a dep resolver, an `onto abandon` command, an `onto set workflow` command, a backward-phase setter (all N2); homonto-engine work; observational setters; any schema redesign beyond the additive `Guides` field.

---

## File Structure

**Layer A — binary (Go, TDD):**
- Modify `internal/ontocli/new.go` — add `--workflow` flag to `newCmd`, thread it into `runNew`, validate against `full|fix|tweak`, stop hardcoding `Workflow: "full"`.
- Modify `internal/ontocli/new_test.go` — tests for each workflow + invalid rejection.
- Modify `internal/ontocli/set.go` — add `baseRefCmd()`, `depsCmd()`, `guidesCmd()`; register them in `setCmd()`.
- Modify `internal/ontocli/set_test.go` — happy-path + reject tests for the three new setters.
- Modify `internal/ontostate/state.go` — add `Guides string` field to `State`, a `validGuides` helper, and a guides shape check in `Validate()`.
- Modify `internal/ontostate/state_test.go` — extend `fullFixtureState()` to set `Guides`; add guides round-trip + shape-reject tests.
- Modify `internal/ontostate/migrate.go` — update the now-stale comment at lines 59–63 (guides is a gated field, not observational; legacy guides value is still not carried). Doc-only; no behavior change.

**Layer B — skills (Markdown, gate-verified):**
- Modify each `catalog/skills/onto{,-open,-design,-build,-verify,-close,-fix,-tweak}/SKILL.md`.
- `catalog/skills/onto-no-slop/SKILL.md` — untouched (prose-only, 0 state refs).
- `catalog/skills/onto/references/*` and every `references/*` file — untouched (they document the schema; excluded from the gate).

**Layer C — enforcement (Bash):**
- Create `scripts/onto-skills-shell-out-check.sh`.
- Modify `scripts/gate.sh` — wire the new check in as a `step`.

### Central design decision (applies to every Layer-B task)

The `onto` binary's forward-only, artifact-gated `onto advance` and terminal `onto close` cannot model three skill operations: (1) **backward phase resets** (mid-build revision `→ design`, verify-fail / reopen `→ build`), (2) **preset phase-skip** (`onto new` always creates `phase: open`, but `onto-fix` / `onto-tweak` operate at build), and (3) the **workflow upgrade** (`fix|tweak → full`). Adding commands for these is N2 (out of scope).

These are handled by the **markdown dispatcher's file-based derivation, which is unchanged** (`onto/SKILL.md` §3: "state.yaml is a cache of truth, not truth" — phase is derived from artifacts; workflow from the proposal's `Preset:` marker). So the Layer-B rewrites **drop the redundant `phase:` / `workflow:` cache writes** (the dispatcher re-derives them) and route the fields that DO have setters (verify-result, decisions, guides, base-ref, deps, close-merged) through the binary. The genuinely un-mapped write — **abandon's `abandoned:` field** (not in the schema) — remains a single documented manual state note, explicitly excepted by the gate. See the Risks section.

---

## Task A1: `onto new --workflow full|fix|tweak`

**Files:**
- Modify: `internal/ontocli/new.go` (`newCmd` at line 42, `runNew` at line 61, the `State` literal at lines 79–84)
- Test: `internal/ontocli/new_test.go`

**Interfaces:**
- Consumes: `ontostate.State`, `ontostate.Save`, `validChangeName`, `gate` (all existing).
- Produces: `onto new <name> [--workflow full|fix|tweak]` writing `onto-state.yaml` with `Workflow` = the flag value (default `full`); invalid values rejected non-zero with no writes.

- [x] **Step 1: Write the failing tests**

Add to `internal/ontocli/new_test.go`:

```go
// TestNewCommand_WorkflowFlag_SetsWorkflow verifies each accepted workflow
// value lands in onto-state.yaml.
func TestNewCommand_WorkflowFlag_SetsWorkflow(t *testing.T) {
	for _, wf := range []string{"full", "fix", "tweak"} {
		t.Run(wf, func(t *testing.T) {
			dir := setUpGatedWorkspace(t)
			if _, err := runOnto(t, "new", "feature-x", "--workflow", wf, "--dir", dir); err != nil {
				t.Fatalf("new --workflow %s: %v", wf, err)
			}
			st, err := ontostate.Load(filepath.Join(dir, "docs", "changes", "feature-x", "onto-state.yaml"))
			if err != nil {
				t.Fatalf("load: %v", err)
			}
			if st.Workflow != wf {
				t.Errorf("Workflow = %q, want %q", st.Workflow, wf)
			}
		})
	}
}

// TestNewCommand_WorkflowDefaultsFull verifies the flag defaults to full.
func TestNewCommand_WorkflowDefaultsFull(t *testing.T) {
	dir := setUpGatedWorkspace(t)
	if _, err := runOnto(t, "new", "feature-y", "--dir", dir); err != nil {
		t.Fatalf("new: %v", err)
	}
	st, _ := ontostate.Load(filepath.Join(dir, "docs", "changes", "feature-y", "onto-state.yaml"))
	if st.Workflow != "full" {
		t.Errorf("Workflow = %q, want full", st.Workflow)
	}
}

// TestNewCommand_InvalidWorkflowCreatesNothing verifies a bad workflow is
// rejected before any write.
func TestNewCommand_InvalidWorkflowCreatesNothing(t *testing.T) {
	dir := setUpGatedWorkspace(t)
	if _, err := runOnto(t, "new", "feature-z", "--workflow", "epic", "--dir", dir); err == nil {
		t.Fatal("new --workflow epic succeeded, want rejection")
	}
	if _, err := os.Stat(filepath.Join(dir, "docs", "changes", "feature-z")); !os.IsNotExist(err) {
		t.Errorf("expected docs/changes/feature-z to not exist, stat err = %v", err)
	}
}
```

`runOnto`, `setUpGatedWorkspace`, `os`, `filepath`, `ontostate` are already available in the package test files.

- [x] **Step 2: Run the tests, watch them fail**

Run: `go test ./internal/ontocli/ -run 'TestNewCommand_(WorkflowFlag|WorkflowDefaultsFull|InvalidWorkflow)' -v`
Expected: FAIL — `--workflow` is an unknown flag (cobra errors), and the invalid-value case does not yet reject.

- [x] **Step 3: Add the flag and validation**

In `internal/ontocli/new.go`, change `newCmd` (lines 42–55) to declare and pass the flag:

```go
func newCmd() *cobra.Command {
	var (
		dir      string
		workflow string
	)

	cmd := &cobra.Command{
		Use:   "new <change-name>",
		Short: "Create a new change-workspace skeleton, if the onto framework is installed",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runNew(cmd, dir, args[0], workflow)
		},
	}
	cmd.Flags().StringVar(&dir, "dir", ".", "workspace root to create the change in")
	cmd.Flags().StringVar(&workflow, "workflow", "full", "workflow for the change: full, fix, or tweak")
	return cmd
}
```

Change `runNew` (line 61) to accept and validate `workflow`, before any `os.MkdirAll`:

```go
func runNew(cmd *cobra.Command, root, name, workflow string) error {
	if err := gate(root); err != nil {
		return err
	}

	if err := validChangeName(name); err != nil {
		return err
	}

	if !ontostate.ValidWorkflow(workflow) {
		return fmt.Errorf("onto new: workflow %q is not one of full|fix|tweak", workflow)
	}
	// ... existing changeDir stat / clobber guard / MkdirAll unchanged ...
```

Set `Workflow: workflow` in the `State` literal (line 81, replacing the hardcoded `"full"`):

```go
	st := ontostate.State{
		Change:   name,
		Workflow: workflow,
		Phase:    "open",
		Created:  time.Now().Format("2006-01-02"),
	}
```

Add the exported membership helper to `internal/ontostate/state.go` (next to `validWorkflows`, line 32) so the CLI validates against the same set and rejects *before* writing (rather than only at `Validate()` after a partial `MkdirAll`):

```go
// ValidWorkflow reports whether w is a recognized workflow value.
func ValidWorkflow(w string) bool { return validWorkflows[w] }
```

- [x] **Step 4: Run the tests, watch them pass**

Run: `go test ./internal/ontocli/ ./internal/ontostate/ -run 'TestNewCommand|ValidWorkflow' -v`
Expected: PASS. Also run the pre-existing `go test ./internal/ontocli/ -run TestNewCommand -v` — the older new-command tests still pass (they never pass `--workflow`, so they get the `full` default).

- [x] **Step 5: Commit**

```bash
git add internal/ontocli/new.go internal/ontocli/new_test.go internal/ontostate/state.go
git commit -m "feat(onto): onto new --workflow full|fix|tweak (default full, reject invalid)"
```

---

## Task A2: `onto set base-ref` and `onto set deps`

**Files:**
- Modify: `internal/ontocli/set.go` (`setCmd` at line 69; add `baseRefCmd`, `depsCmd` alongside `closeMergedCmd`/`directiveCmd`)
- Test: `internal/ontocli/set_test.go`

**Interfaces:**
- Consumes: `runTransition` (line 15) — the load→apply→Validate→Save helper that writes nothing on failure.
- Produces:
  - `onto set base-ref <change> <ref>` — presence-only; empty ref rejected; sets `State.BaseRef`.
  - `onto set deps <change> --dep <name> [--dep <name> ...]` — repeatable flag; sets `State.Deps` to the collected slice (may be empty to clear).

- [x] **Step 1: Write the failing tests**

Add to `internal/ontocli/set_test.go`:

```go
func TestSetBaseRef_HappyPath_WritesField(t *testing.T) {
	root := prepWorkspace(t)
	seedChange(t, root, "c", "open")

	if _, err := runOnto(t, "set", "base-ref", "c", "abc123", "--dir", root); err != nil {
		t.Fatalf("set base-ref: %v", err)
	}
	st, _ := ontostate.LoadChange(filepath.Join(root, "docs", "changes", "c"))
	if st.BaseRef != "abc123" {
		t.Errorf("BaseRef = %q, want abc123", st.BaseRef)
	}
}

func TestSetBaseRef_EmptyRejected(t *testing.T) {
	root := prepWorkspace(t)
	seedChange(t, root, "c", "open")

	if _, err := runOnto(t, "set", "base-ref", "c", "", "--dir", root); err == nil {
		t.Fatal("empty base-ref accepted, want rejection")
	}
	st, _ := ontostate.LoadChange(filepath.Join(root, "docs", "changes", "c"))
	if st.BaseRef != "" {
		t.Errorf("BaseRef = %q, want unchanged empty", st.BaseRef)
	}
}

func TestSetDeps_HappyPath_CollectsRepeatedFlag(t *testing.T) {
	root := prepWorkspace(t)
	seedChange(t, root, "c", "open")

	if _, err := runOnto(t, "set", "deps", "c", "--dep", "dep-a", "--dep", "dep-b", "--dir", root); err != nil {
		t.Fatalf("set deps: %v", err)
	}
	st, _ := ontostate.LoadChange(filepath.Join(root, "docs", "changes", "c"))
	if !reflect.DeepEqual(st.Deps, []string{"dep-a", "dep-b"}) {
		t.Errorf("Deps = %v, want [dep-a dep-b]", st.Deps)
	}
}
```

Add `"reflect"` to the `set_test.go` import block.

- [x] **Step 2: Run the tests, watch them fail**

Run: `go test ./internal/ontocli/ -run 'TestSetBaseRef|TestSetDeps' -v`
Expected: FAIL — `base-ref` and `deps` are unknown subcommands of `onto set`.

- [x] **Step 3: Add the two setters**

In `internal/ontocli/set.go`, register them in `setCmd()` (after line 85, before `return cmd`):

```go
	cmd.AddCommand(baseRefCmd())
	cmd.AddCommand(depsCmd())
```

Add the two command constructors (mirroring the existing `directiveCmd` shape):

```go
// baseRefCmd records the change's base ref verbatim; presence-only shape
// (empty rejected — a base ref is a real commit reference, not a clear).
func baseRefCmd() *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:   "base-ref <change> <ref>",
		Short: "Record the base git ref a change branched from",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, ref := args[0], args[1]
			return runTransition(cmd, dir, name, func(st *ontostate.State) error {
				if ref == "" {
					return fmt.Errorf("onto set base-ref: ref must not be empty")
				}
				st.BaseRef = ref
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&dir, "dir", ".", "workspace root containing the change")
	return cmd
}

// depsCmd sets the change's dependency list from a repeatable --dep flag.
// --dep is used (not a comma-split positional) so dependency names carrying
// edge characters are never ambiguously parsed.
func depsCmd() *cobra.Command {
	var (
		dir  string
		deps []string
	)
	cmd := &cobra.Command{
		Use:   "deps <change> --dep <name> [--dep <name> ...]",
		Short: "Set a change's dependency list (repeat --dep per dependency)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTransition(cmd, dir, args[0], func(st *ontostate.State) error {
				st.Deps = deps
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&dir, "dir", ".", "workspace root containing the change")
	cmd.Flags().StringArrayVar(&deps, "dep", nil, "a dependency change name; repeat for several")
	return cmd
}
```

- [x] **Step 4: Run the tests, watch them pass**

Run: `go test ./internal/ontocli/ -run 'TestSetBaseRef|TestSetDeps' -v`
Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add internal/ontocli/set.go internal/ontocli/set_test.go
git commit -m "feat(onto): onto set base-ref and onto set deps (repeatable --dep)"
```

---

## Task A3: `Guides` gated field + `onto set guides`

**Files:**
- Modify: `internal/ontostate/state.go` (`State` struct at lines 62–82; `Validate()` at lines 117–143)
- Modify: `internal/ontostate/migrate.go` (comment at lines 59–63 — doc only)
- Modify: `internal/ontostate/state_test.go` (`fullFixtureState()` at line ~438; add guides tests)
- Modify: `internal/ontocli/set.go` (`setCmd` registration; add `guidesCmd`)
- Test: `internal/ontocli/set_test.go`

**Interfaces:**
- Consumes: `runTransition`.
- Produces:
  - `State.Guides string` (yaml `guides,omitempty`) with shape `"" | "pending" | "updated" | "waived:<...>"`.
  - `ontostate.ValidGuides(v string) bool`.
  - `onto set guides <change> <value>` — custom validator (not `enumSetterCmd`, because `waived:` is a prefix, not a fixed member).

**Note on existing DeepEqual tests:** adding `Guides` with `omitempty` is additive-safe. Every existing round-trip / migration `reflect.DeepEqual` test constructs a `State` (or produces one via `migrateLegacy`) with `Guides == ""` on both `got` and `want`, so none breaks. Step 5 below deliberately extends `fullFixtureState()` to set `Guides` so the round-trip and migrate-idempotency tests actively exercise the new field.

- [x] **Step 1: Write the failing state-model tests**

Add to `internal/ontostate/state_test.go`:

```go
func TestValidate_Guides_AcceptsAllowedShapes(t *testing.T) {
	for _, g := range []string{"", "pending", "updated", "waived: no user-facing surface"} {
		st := State{Change: "c", Phase: "close", Guides: g}
		if err := st.Validate(); err != nil {
			t.Errorf("Validate() with guides %q = %v, want nil", g, err)
		}
	}
}

func TestValidate_Guides_RejectsUnknown(t *testing.T) {
	st := State{Change: "c", Phase: "close", Guides: "done"}
	if err := st.Validate(); err == nil {
		t.Fatal("Validate() with guides \"done\" = nil, want error")
	}
}
```

- [x] **Step 2: Run, watch them fail**

Run: `go test ./internal/ontostate/ -run 'TestValidate_Guides' -v`
Expected: FAIL — `Guides` is not a field yet (compile error), so add the field first is required to even compile; the reject test then fails until validation exists.

- [x] **Step 3: Add the field, helper, and validation**

In `internal/ontostate/state.go`, add `Guides` to the gated core of `State` (after `Directive`, line 77):

```go
	Directive string   `yaml:"directive,omitempty" json:"directive,omitempty"`
	Guides    string   `yaml:"guides,omitempty" json:"guides,omitempty"` // "" | pending | updated | waived:<reason>
	Archived  bool     `yaml:"archived,omitempty" json:"archived,omitempty"`
```

Add the helper (near the other membership sets, after line 38):

```go
// ValidGuides reports whether v is a recognized guides value: empty (unset),
// "pending", "updated", or any "waived:<reason>". The waived form is a prefix,
// not a fixed member, so guides cannot use the enum-setter machinery.
func ValidGuides(v string) bool {
	return v == "" || v == "pending" || v == "updated" || strings.HasPrefix(v, "waived:")
}
```

`strings` is already imported in `state.go`.

Add the check to `Validate()` (after the `verify.result` check, before `return nil` at line 141):

```go
	if !ValidGuides(s.Guides) {
		return fmt.Errorf("onto-state: guides %q is not one of pending|updated|waived:<reason>", s.Guides)
	}
```

- [x] **Step 4: Run, watch the state-model tests pass**

Run: `go test ./internal/ontostate/ -run 'TestValidate_Guides' -v`
Expected: PASS.

- [x] **Step 5: Extend the full fixture and fix the stale migrate comment**

In `internal/ontostate/state_test.go`, add `Guides` to `fullFixtureState()` (after the `Directive` line):

```go
		Directive:     "user said: ship it without asking again",
		Guides:        "updated",
		Archived:      false,
```

In `internal/ontostate/migrate.go`, replace the stale comment on `migrateLegacy` (lines 62–63) so it no longer calls guides "observational-only", while preserving the actual behavior (legacy guides value is still not carried — it re-resolves at close, and empty is a valid guides shape):

```go
// Renames: execution->build_mode, tdd->tdd_mode, verify.mode->verify.scale,
// metrics.upgraded->preset_escalated. "guides" is now a gated field but a
// legacy file's guides value is intentionally not carried: it re-resolves at
// close (onto-close sets it), and empty is a valid guides shape.
```

- [x] **Step 6: Run the full state package, watch the DeepEqual fixtures still pass**

Run: `go test ./internal/ontostate/ -count=1`
Expected: PASS — `fullFixtureState()`-based round-trip and `TestParseAndMigrate_CurrentVersion_IsNoOp` now carry `Guides: "updated"` on both sides and remain equal; migration tests that build their own `want` (no guides) still match `migrateLegacy` output (`Guides == ""`).

- [x] **Step 7: Write the failing setter test**

Add to `internal/ontocli/set_test.go`:

```go
func TestSetGuides_HappyPaths(t *testing.T) {
	root := prepWorkspace(t)
	seedChange(t, root, "c", "close")

	for _, g := range []string{"pending", "updated", "waived: no user-facing surface"} {
		if _, err := runOnto(t, "set", "guides", "c", g, "--dir", root); err != nil {
			t.Fatalf("set guides %q: %v", g, err)
		}
		st, _ := ontostate.LoadChange(filepath.Join(root, "docs", "changes", "c"))
		if st.Guides != g {
			t.Errorf("Guides = %q, want %q", st.Guides, g)
		}
	}
}

func TestSetGuides_BadValueRejectedNoWrite(t *testing.T) {
	root := prepWorkspace(t)
	seedChange(t, root, "c", "close")

	if _, err := runOnto(t, "set", "guides", "c", "done", "--dir", root); err == nil {
		t.Fatal("set guides done accepted, want rejection")
	}
	st, _ := ontostate.LoadChange(filepath.Join(root, "docs", "changes", "c"))
	if st.Guides != "" {
		t.Errorf("Guides = %q, want unchanged empty", st.Guides)
	}
}
```

- [x] **Step 8: Run, watch it fail**

Run: `go test ./internal/ontocli/ -run 'TestSetGuides' -v`
Expected: FAIL — `guides` is not a subcommand of `onto set`.

- [x] **Step 9: Add `guidesCmd`**

In `internal/ontocli/set.go`, register it in `setCmd()` (after the `depsCmd()` line from Task A2):

```go
	cmd.AddCommand(guidesCmd())
```

Add the constructor (custom validator, not `enumSetterCmd`, because of the `waived:` prefix):

```go
// guidesCmd sets the guides obligation field. It cannot use enumSetterCmd
// because the "waived:<reason>" form is a prefix, not a fixed enum member.
func guidesCmd() *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:   "guides <change> <value>",
		Short: "Set a change's guides obligation: pending, updated, or waived:<reason>",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, value := args[0], args[1]
			return runTransition(cmd, dir, name, func(st *ontostate.State) error {
				if !ontostate.ValidGuides(value) || value == "" {
					return fmt.Errorf("onto set guides: %q is not one of pending|updated|waived:<reason>", value)
				}
				st.Guides = value
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&dir, "dir", ".", "workspace root containing the change")
	return cmd
}
```

(The `value == ""` guard rejects an empty *setter* argument — empty is a valid stored shape but not a meaningful thing to *set*, matching `directiveCmd`'s non-empty policy.)

- [x] **Step 10: Run, watch it pass**

Run: `go test ./internal/ontocli/ -run 'TestSetGuides' -v`
Expected: PASS.

- [x] **Step 11: Commit**

```bash
git add internal/ontostate/state.go internal/ontostate/migrate.go internal/ontostate/state_test.go internal/ontocli/set.go internal/ontocli/set_test.go
git commit -m "feat(onto): add guides gated field + onto set guides (no schema bump)"
```

---

## Layer B — rewrite the eight skills to shell out

Every Layer-B task edits Markdown only. After each task, run the grep gate from Task C1 *once it exists* (Task C1 is authored first so it can prove-fail before these rewrites). Until C1 exists, verify each task by re-reading the file. Apply the **field-name mapping** from Global Constraints and the **central design decision** (drop derivation-covered `phase:`/`workflow:` cache writes; route setter-backed fields through the binary; remove metric writes).

### Field→command map (from the Design Doc, the reference for every rewrite)

| Skill instruction (state write) | Replacement |
|---|---|
| create `state.yaml` (change/workflow/phase:open/created) | `onto new <name> --workflow <full\|fix\|tweak>` |
| set `base_ref` | `onto set base-ref <name> "$(git rev-parse HEAD)"` |
| set `deps` from `Depends-on:` | `onto set deps <name> --dep <a> --dep <b>` |
| advance phase (forward, full workflow) | `onto advance <name>` |
| set `decisions.isolation` | `onto set isolation <name> <branch\|worktree>` |
| set `decisions.execution` | `onto set build-mode <name> <direct\|subagent>` |
| set `decisions.tdd` | `onto set tdd-mode <name> <tdd\|direct>` |
| set `decisions.directive` | `onto set directive <name> "<text>"` |
| set `verify.mode` | `onto set verify-scale <name> <light\|full>` |
| set `verify.result` | `onto set verify-result <name> <pending\|pass\|fail>` |
| set `close.merged: true` | `onto set close-merged <name>` |
| set `guides` | `onto set guides <name> <pending\|updated\|"waived: <reason>">` |
| archive (`git mv` + `archived: true`) | `onto close <name>` (moves to `archive/` and sets `archived`) |
| read a state field value | `onto state <name> --json` |
| `metrics.*` (phases, verify_rounds, upgraded) | **removed** — no longer written |
| backward `phase:` (revision/reopen), `workflow:` upgrade | **dropped** — dispatcher re-derives from artifacts/`Preset:` marker |
| `abandoned:` + `archived: true` (abandon) | documented manual write, gate-excepted (no command; N2) |

---

## Task B1a: rewrite `onto-open`

**Files:** Modify `catalog/skills/onto-open/SKILL.md`

- [x] **Step 1: Rewrite step 3 "Create the workspace"** (lines 58–76). Replace the `state.yaml` bullet with a command sequence. New text for the `state.yaml`/workspace bullets:

```markdown
- Create the workspace via the binary: `onto new <name> --workflow full`
  (writes `onto-state.yaml` with `change`, `workflow: full`, `phase: open`,
  `created`; and empty `proposal.md`/`tasks.md`). Then record the creation
  fields the same way:
  - `onto set base-ref <name> "$(git rev-parse HEAD)"` — captured NOW, before
    anything is committed; written once, never recomputed.
  - `onto set deps <name> --dep <a> --dep <b>` for each `Depends-on:` entry
    (omit entirely when there are none).
- `notes.md` — template: `references/notes.md`. Created NOW, seeded with the
  confirmed clarification summary. (unchanged)
- `proposal.md` — template: `references/proposal.md`; fill the skeleton `onto
  new` created. (unchanged)
- `tasks.md` — template: `references/tasks.md`; fill the skeleton. (unchanged)
```

Remove the "decisions null, metrics initialized per the template: `phases: {}`, counters 0, `upgraded: false`" clause entirely (metrics dropped; decisions default to empty in the binary).

- [x] **Step 2: Rewrite the exit checklist** (lines 83–104). Replace the phase-advance and metrics lines:

- Change `- [ ] `state.yaml` phase advanced: `open → design` — written only after ...` to:

```markdown
- [x] Phase advanced open → design via `onto advance <name>` — run **only
      after** the artifact-review gate is answered, never before (the
      dispatcher treats a lagging phase as an unanswered gate and re-presents it)
```

- Delete the line `- [ ] `metrics.phases.open: <today>` stamped` entirely.

- [x] **Step 3: Verify no state-write phrasing remains**

Run: `grep -nE 'state\.yaml|metrics\.' catalog/skills/onto-open/SKILL.md`
Expected: no line instructs *writing* `state.yaml` and no `metrics.` reference remains (a residual read reference, if any, is acceptable; there should be none in onto-open).

- [x] **Step 4: Commit**

```bash
git add catalog/skills/onto-open/SKILL.md
git commit -m "docs(onto): onto-open shells out to the binary for state writes"
```

---

## Task B1b: rewrite `onto-design`

**Files:** Modify `catalog/skills/onto-design/SKILL.md`

- [x] **Step 1: Rewrite the exit checklist** (lines 111–112). Replace:

```markdown
- [x] `state.yaml` phase advanced: `design → build`;
      `metrics.phases.design: <today>` stamped
```

with:

```markdown
- [x] Phase advanced design → build via `onto advance <name>`
```

- [x] **Step 2: Verify**

Run: `grep -nE 'state\.yaml|metrics\.' catalog/skills/onto-design/SKILL.md`
Expected: no write instruction, no `metrics.` reference.

- [x] **Step 3: Commit**

```bash
git add catalog/skills/onto-design/SKILL.md
git commit -m "docs(onto): onto-design shells out for the phase advance"
```

---

## Task B1c: rewrite `onto-build`

**Files:** Modify `catalog/skills/onto-build/SKILL.md`

- [x] **Step 1: Rewrite the plan-ready gate config recording** (lines 41–48). Replace "recorded in `state.yaml` under `decisions:`" and the three bullets so each decision is recorded through the binary:

```markdown
> **GATE (plan-ready + execution config):** pause. The user reviews the plan
> and chooses the execution configuration, recorded through the binary:
>
> - `onto set isolation <name> branch|worktree` — branch for simple changes;
>   worktree for parallel work or a dirty current branch
> - `onto set build-mode <name> direct|subagent` — direct in-session; subagent
>   only when real background dispatch capability exists
> - `onto set tdd-mode <name> tdd|direct` — tdd for anything with testable
>   logic; direct for content/docs deliverables
```

- [x] **Step 2: Rewrite the directive-recording sentence** (lines 53–55). Replace "record it **verbatim** in `decisions.directive`" with "record it **verbatim** via `onto set directive <name> \"<text>\"`".

- [x] **Step 3: Rewrite the mid-build medium-revision sequence** (lines 111–123). The `design.md` and `verification.md` `Status:`/`Result:` flips are prose/artifact edits and stay. Replace only the two *state* writes:
  - "(2) ... set `state.yaml` `verify.result: pending`" → "(2) ... run `onto set verify-result <name> pending`".
  - "(3) set `phase: design`" → drop the state write; replace with: "(3) the `Status: Under revision` marker now drives the dispatcher's derivation to `design` (files win downward) — no phase field is written; the next dispatch routes to design."

Keep the surrounding ordering rationale intact.

- [x] **Step 4: Rewrite the exit checklist** (lines 138–139). Replace:

```markdown
- [x] `decisions:` in `state.yaml` filled (isolation, execution, tdd)
- [x] `state.yaml` phase advanced: `build → verify`;
      `metrics.phases.build: <today>` stamped
```

with:

```markdown
- [x] Decisions recorded via `onto set isolation|build-mode|tdd-mode <name> …`
- [x] Phase advanced build → verify via `onto advance <name>`
```

- [x] **Step 5: Verify**

Run: `grep -nE 'state\.yaml|metrics\.|decisions\.' catalog/skills/onto-build/SKILL.md`
Expected: no write instruction; a surviving mention of "files win downward" derivation is fine and names no state file.

- [x] **Step 6: Commit**

```bash
git add catalog/skills/onto-build/SKILL.md
git commit -m "docs(onto): onto-build shells out for decisions, advance, revision"
```

---

## Task B1d: rewrite `onto-verify`

**Files:** Modify `catalog/skills/onto-verify/SKILL.md`

- [x] **Step 1: Rewrite step 1 scale check** (line 24). Replace "Set `verify.mode` in `state.yaml`:" with "Set the verification scale via `onto set verify-scale <name> light|full`:". Keep the two bullets describing full vs light.

- [x] **Step 2: Remove the metric increment in the adversarial pass** (line 61). Delete "Increment `metrics.verify_rounds` once per round." — round counting is durable in `notes.md` (the skill already says notes, not metrics, is the durable counter).

- [x] **Step 3: Rewrite step 4 result mirror** (line 78). Replace "Mirror the result into `state.yaml` `verify.result`." with "Record the result via `onto set verify-result <name> pass|fail`."

- [x] **Step 4: Rewrite the failure gate** (lines 82–92). The `verification.md` edits stay. Replace the state writes:
  - "→ back to build: reset `phase: build`, add tasks for the fixes" → "→ back to build: add tasks for the fixes in `tasks.md`; the unchecked tasks drive the dispatcher's derivation back to build (files win downward) — no phase field is written".
  - The `Result:`/`verify.result` "stay `pass`" clause: change "`verify.result` stay `pass`" to "run `onto set verify-result <name> pass` (accepted deviations recorded in `verification.md`)".

- [x] **Step 5: Rewrite the exit checklist** (lines 99–109). Replace:
  - "`verify.result: pass` in both the report and `state.yaml`" → "`verify.result: pass` recorded via `onto set verify-result <name> pass` and in the report".
  - "`metrics.verify_rounds` incremented **once** ... (this checklist owns the increment ...)" → delete the metric-increment clause; keep "Adversarial pass run (or its skip recorded in the report's Adversarial section)".
  - "`state.yaml` phase advanced: `verify → close`; `metrics.phases.verify: <today>` stamped" → "Phase advanced verify → close via `onto advance <name>`".

- [x] **Step 6: Verify**

Run: `grep -nE 'state\.yaml|metrics\.' catalog/skills/onto-verify/SKILL.md`
Expected: no write instruction, no `metrics.` reference.

- [x] **Step 7: Commit**

```bash
git add catalog/skills/onto-verify/SKILL.md
git commit -m "docs(onto): onto-verify shells out for scale/result/advance, drops metrics"
```

---

## Task B1e: rewrite `onto-close`

**Files:** Modify `catalog/skills/onto-close/SKILL.md`

- [x] **Step 1: Rewrite the idempotency re-entry check** (lines 17–22). Replace "If `state.yaml` shows `close.merged: true`" with "If `onto state <name> --json` shows `close.merged: true` (read it at entry)".

- [x] **Step 2: Rewrite the guides obligation** (lines 48–54). Replace the parenthetical and the two outcomes:
  - "(`guides:` in `state.yaml`)" → "(read via `onto state <name> --json`)".
  - "→ `guides: updated`" → "then `onto set guides <name> updated`".
  - "record `guides: \"waived: <reason>\"` (quoted — a bare ... invalid YAML ...)" → "record `onto set guides <name> \"waived: <reason>\"` (the reason comes from the user or a recorded directive, never invented)". Drop the YAML-quoting note (the binary owns serialization now).

- [x] **Step 3: Rewrite step 3.1 merged flag** (lines 76–78). Replace "Set `close.merged: true` in `state.yaml` **before merging**" with "Run `onto set close-merged <name>` **before merging**".

- [x] **Step 4: Remove step 3.5 metrics finalization** (lines 118–119). Delete the "Finalize `metrics`: `phases.close: <today>`, `tasks_total`, `verify_rounds`, `upgraded`. Observational — never block on them." item and renumber the following archive step from 6 to 5.

- [x] **Step 5: Rewrite step 3.6 → 3.5 archive** (lines 120–127). Replace the manual `git mv` + `archived: true` + one-commit instruction with the binary command:

```markdown
5. **Archive via the binary**: `onto close <name>` — it verifies the change is
   at `close`, all `deps` are archived, and the worktree is clean, then moves
   `docs/changes/<name>` to `docs/changes/archive/YYYY-MM-DD-<name>` and sets
   `archived: true` in one operation. Commit the move (`git add -A && git
   commit`). `phase` stays `close`; "done" is derived-only, never written. The
   archived workspace is history — never edited after, with one sanctioned
   exception: `ship.md`.
```

- [x] **Step 6: Rewrite the exit checklist** (lines 141–151):
  - "`close.merged: true` set before the merge" → "`onto set close-merged` run before the merge".
  - "`guides: updated` or `guides: \"waived: <reason>\"`" → "`onto set guides <name> updated` or `… \"waived: <reason>\"` — never pending".
  - Delete the "Metrics finalized (phase dates, tasks_total, verify_rounds, upgraded)" item.

- [x] **Step 7: Verify**

Run: `grep -nE 'state\.yaml|metrics\.' catalog/skills/onto-close/SKILL.md`
Expected: no write instruction, no `metrics.` reference (reads route through `onto state --json`).

- [x] **Step 8: Commit**

```bash
git add catalog/skills/onto-close/SKILL.md
git commit -m "docs(onto): onto-close shells out for guides/close-merged/archive, drops metrics"
```

---

## Task B1f: rewrite `onto-fix`

**Files:** Modify `catalog/skills/onto-fix/SKILL.md`

- [x] **Step 1: Rewrite step 1 open-lite workspace creation** (lines 37–60). Replace the `state.yaml` bullet:

```markdown
- Create the workspace via `onto new <name> --workflow fix` (writes
  `onto-state.yaml` with `workflow: fix`, `phase: open`, `created`, and empty
  `proposal.md`/`tasks.md`). Then:
  - `onto set base-ref <name> "$(git rev-parse HEAD)"`
  - `onto set guides <name> pending`
  - default the decisions (presets enter build directly): `onto set isolation
    <name> branch`, `onto set build-mode <name> direct`, **`onto set tdd-mode
    <name> tdd`** — a fix's whole method is a failing test that reproduces the
    bug first, so its build runs the TDD branch; never default a fix to
    `tdd-mode direct`.
- `proposal.md` — a `Preset: fix` line at column 0 under the title ... (unchanged)
- `tasks.md` — short checklist (reproduce → fix → regression). (unchanged)
```

Add a note after the bullet, replacing the "Stamp `metrics.phases.<phase>` ..." sentence (lines 58–61): "`onto new` records `phase: open`; the preset skips design, so its working phase (build) is **derived** by the dispatcher (`workflow: fix` + workspace → build). The binary's `phase` field is not advanced through the skipped phases — that reconciliation is out of scope (N2)." Delete the metrics-stamp sentence.

- [x] **Step 2: Rewrite the upgrade gate** (lines 105–112). Replace "set `workflow: full`, `phase: design`, `metrics.upgraded: true` in `state.yaml`, **and annotate the proposal's first line to `Preset: fix (upgraded to full YYYY-MM-DD)`**" with:

```markdown
> On confirmed upgrade: **annotate the proposal's first line to `Preset: fix
> (upgraded to full YYYY-MM-DD)`** — the dispatcher re-derives `workflow: full`
> from that marker (there is no `onto set workflow`; the marker is the
> authority the state-rebuild reads). Then run `onto advance <name>` to reach
> design and route through `/onto` to backfill it. Never keep patching past a
> trigger "because it's almost done".
```

Drop `metrics.upgraded: true`.

- [x] **Step 3: Rewrite the exit checklist** (lines 114–125): replace "`verify.result` set" with "`verify.result` set via `onto set verify-result`". No metrics references exist in this checklist to remove.

- [x] **Step 4: Verify**

Run: `grep -nE 'state\.yaml|metrics\.' catalog/skills/onto-fix/SKILL.md`
Expected: no write instruction, no `metrics.` reference.

- [x] **Step 5: Commit**

```bash
git add catalog/skills/onto-fix/SKILL.md
git commit -m "docs(onto): onto-fix shells out; preset phase stays derivation-driven"
```

---

## Task B1g: rewrite `onto-tweak`

**Files:** Modify `catalog/skills/onto-tweak/SKILL.md`

- [x] **Step 1: Rewrite step 1 open-lite** (lines 38–48). Replace the `state.yaml` clause:

```markdown
One-paragraph `proposal.md` — a `Preset: tweak` line at column 0 under the
title, then what + why — plus short `tasks.md`. Create the workspace via
`onto new <name> --workflow tweak`, then `onto set base-ref <name> "$(git
rev-parse HEAD)"`, `onto set guides <name> pending`, and the default decisions:
`onto set isolation <name> branch`, `onto set build-mode <name> direct`, `onto
set tdd-mode <name> direct`. Branch: `tweak/YYYYMMDD/<name>`. **Commit the
workspace** before the first task. `onto new` records `phase: open`; the
preset's working phase (build) is derived by the dispatcher — the binary's
`phase` field is not advanced through the skipped phases (N2, out of scope).
```

Delete the "Stamp `metrics.phases.<phase>` ..." sentence (lines 46–48).

- [x] **Step 2: Rewrite the upgrade gate** (lines 88–96). Replace "set `workflow: full`, `phase: design`, `metrics.upgraded: true`, **and annotate the proposal's first line to `Preset: tweak (upgraded to full YYYY-MM-DD)`**" with:

```markdown
> On confirmed upgrade: **annotate the proposal's first line to `Preset: tweak
> (upgraded to full YYYY-MM-DD)`** — the dispatcher re-derives `workflow: full`
> from that marker (no `onto set workflow` exists). Then run `onto advance
> <name>` to reach design and route through `/onto` to backfill it.
```

- [x] **Step 3: Verify**

Run: `grep -nE 'state\.yaml|metrics\.' catalog/skills/onto-tweak/SKILL.md`
Expected: no write instruction, no `metrics.` reference.

- [x] **Step 4: Commit**

```bash
git add catalog/skills/onto-tweak/SKILL.md
git commit -m "docs(onto): onto-tweak shells out; preset phase stays derivation-driven"
```

---

## Task B2: delete the "markdown-only / no external CLI" copy from `onto/SKILL.md`

**Files:** Modify `catalog/skills/onto/SKILL.md`

The no-CLI copy lives only in `onto/SKILL.md` (grep confirmed: lines 8, 13–14; the `references/lint-checklist.md` "no scripts" is a different, excluded context — leave it). This task also rewrites the dispatcher's own state-write mentions (abandon, reopen, directive-record) per the central design decision.

- [x] **Step 1: Rewrite the intro paragraph** (lines 8–14). Replace:

```markdown
onto is a self-contained, markdown-only development workflow. Five phases —
**open → design → build → verify → close** — plus two preset paths
(`onto-fix` for bugs, `onto-tweak` for small non-bug changes). All artifacts
live in one `docs/` tree; phase state lives in an agent-managed
`docs/changes/<name>/state.yaml` that is always cross-checked against real
file state. There are no scripts and no external workflow CLIs: the skills
are the machinery.
```

with:

```markdown
onto is a five-phase development workflow — **open → design → build → verify →
close** — plus two preset paths (`onto-fix` for bugs, `onto-tweak` for small
non-bug changes). All artifacts live in one `docs/` tree. **Every state
mutation goes through the `onto` binary** (`onto new`, `onto set …`, `onto
advance`, `onto close`): it is the single authority for `onto-state.yaml` and a
hard dependency of these skills — the tooling preflight below fails loudly if it
is missing. The skills never hand-edit the state file. Phase is always
cross-checked against real file state: the state file is a cache of truth, not
truth.
```

- [x] **Step 2: Add `onto` to the tooling preflight as a hard dependency.** In section 1 (lines 19–46), the preflight currently warns-never-halts for `rtk`/`graphify`. Add a first check that the `onto` binary is on PATH and, unlike the others, is required:

```markdown
0. **onto binary** (required — this is the one hard dependency). Run `onto
   version`. On failure, STOP: the skills drive all workflow state through the
   `onto` binary; without it no phase can mutate state safely. Tell the user to
   install/build it (`go build ./cmd/onto`) before proceeding. This is the only
   preflight check that halts; `rtk` and `graphify` below still warn-never-halt.
```

- [x] **Step 3: Rewrite the directive-record sentence** (line 231). Replace "must be recorded verbatim in `decisions.directive` in `state.yaml`" with "must be recorded verbatim via `onto set directive <name> \"<text>\"`".

- [x] **Step 4: Rewrite the reopen path** (lines 194–200). The `verification.md` `Result:` flip stays. Replace "Reset `phase: build`, add tasks for the fix, and flip ... with `state.yaml verify.result: pending`" with: "Add tasks for the fix in `tasks.md` and run `onto set verify-result <name> pending`; flip `verification.md`'s `Result:` line to `Result: superseded (reopened <date>)`. The unchecked tasks plus the invalidated result drive the dispatcher's derivation back to build — no phase field is written."

- [x] **Step 5: Rewrite the abandon path** (lines 201–205) — the one gate-excepted manual write. Keep it explicit and flagged:

```markdown
- **Abandon** — the user drops a change: there is no `onto abandon` command
  (deferred to N2), so this is the single sanctioned direct state note. Add
  `abandoned: "<reason>"` (the user's words) to `onto-state.yaml`, then `onto
  close <name>` to move the workspace to `docs/changes/archive/YYYY-MM-DD-<name>/`
  and set `archived: true`, in one commit. It leaves the active list and never
  routes anywhere again. No spec merge, no ADR numbering.
```

(`onto close` requires `phase: close`; an abandoned change may be at any phase. If `onto close` refuses, fall back to the manual `git mv` + `archived: true` note — record which was used. This residual is the flagged N2 gap.)

- [x] **Step 6: Verify the no-CLI copy is gone and the intro states the dependency**

Run: `grep -niE 'markdown-only|no scripts and no external|are the machinery' catalog/skills/onto/SKILL.md`
Expected: no matches.

- [x] **Step 7: Commit**

```bash
git add catalog/skills/onto/SKILL.md
git commit -m "docs(onto): drop markdown-only/no-CLI copy; state the hard onto binary dependency"
```

---

## Task C1: the grep enforcement gate

**Files:**
- Create: `scripts/onto-skills-shell-out-check.sh`
- Modify: `scripts/gate.sh`

**Interfaces:**
- Produces: an executable script exiting non-zero if any of the eight `catalog/skills/onto{,-*}/SKILL.md` files (excluding `onto-no-slop` and every `references/` file) contains a direct state-file write instruction or the markdown-only/no-CLI copy.

### Gate scope — include / exclude (the flagged-precision decision)

- **Include:** exactly the eight `SKILL.md` files: `catalog/skills/onto/SKILL.md` and `catalog/skills/onto-{open,design,build,verify,close,fix,tweak}/SKILL.md`.
- **Exclude:** `catalog/skills/onto-no-slop/**` (prose-only); **every `references/` file** — `catalog/skills/onto/references/state-yaml.md` legitimately *documents* the schema and mentions `state.yaml` throughout, and the other `references/*.md` are templates/checklists. The gate never reads under `references/`.
- **The gate is a curated blocklist, not a token ban.** It cannot ban the bare token `state.yaml`, because the dispatcher's discovery and derivation legitimately *read* the state file ("Scan `docs/changes/*/` … holds a `proposal.md` or a `state.yaml`", "reads `archived: false`"). The blocklist targets **mutation phrasings**, **metric writes**, and the **no-CLI copy** — the exact strings the Layer-B rewrites remove. Its fragility (a future write phrased outside the blocklist) is the accepted limit; it guards the "skill hand-edits state" regression class, the same coarse-but-real guarantee `spec-command-check.sh` gives.
- **One sanctioned exception:** the abandon path in `onto/SKILL.md` writes `abandoned: "<reason>"` directly (no command exists; N2). The blocklist does not match `abandoned:` (a field with no setter), so this survives without a special-case allowlist.

- [x] **Step 1: Author the check script (before the rewrites land, so it can prove-fail)**

Create `scripts/onto-skills-shell-out-check.sh`:

```bash
#!/usr/bin/env bash
# Enforces onto-skills-shell-out: the onto* SKILL.md files must drive every
# state mutation through the `onto` binary — never a direct state-file write —
# and must not carry the retired "markdown-only / no external CLI" copy.
#
# Deliberately coarse (like spec-command-check.sh): a curated blocklist of
# mutation phrasings, metric-write references, and the no-CLI copy. It guards
# the "a skill hand-edits state.yaml" regression class, not full semantics.
# Scope is the eight SKILL.md files only; references/ (which DOCUMENTS the
# schema, e.g. onto/references/state-yaml.md) and onto-no-slop are excluded.
set -euo pipefail
cd "$(dirname "$0")/.."

FILES=(
  catalog/skills/onto/SKILL.md
  catalog/skills/onto-open/SKILL.md
  catalog/skills/onto-design/SKILL.md
  catalog/skills/onto-build/SKILL.md
  catalog/skills/onto-verify/SKILL.md
  catalog/skills/onto-close/SKILL.md
  catalog/skills/onto-fix/SKILL.md
  catalog/skills/onto-tweak/SKILL.md
)

# 1. The retired no-CLI / markdown-only copy (any of these substrings).
NOCLI='markdown-only|no scripts and no external|are the machinery'

# 2. Metric-write references (all removed by the rewrite).
METRICS='metrics\.(phases|verify_rounds|upgraded)'

# 3. Direct state-file mutation phrasings: a mutation cue on the same line as a
#    state-file token. The abandon `abandoned:` field (no setter, N2) is not a
#    cue and is intentionally not matched.
MUTATE='(set|write|record|stamp|mirror|flip|reset|fill|filled|initialize[d]?|phase advanced|advanced:)[^\n]*(state\.yaml|onto-state\.yaml)|(state\.yaml|onto-state\.yaml)[^\n]*(phase advanced|verify\.result|close\.merged: true|decisions:|guides:)'

fails=0
for f in "${FILES[@]}"; do
  if hits="$(grep -nEi "$NOCLI" "$f")"; then
    printf 'FAIL: %s contains retired no-CLI/markdown-only copy:\n%s\n' "$f" "$hits"
    fails=$((fails + 1))
  fi
  if hits="$(grep -nE "$METRICS" "$f")"; then
    printf 'FAIL: %s contains a metric-write reference (metrics are dropped):\n%s\n' "$f" "$hits"
    fails=$((fails + 1))
  fi
  if hits="$(grep -nEi "$MUTATE" "$f")"; then
    printf 'FAIL: %s contains a direct state-file write instruction:\n%s\n' "$f" "$hits"
    fails=$((fails + 1))
  fi
done

if [ "$fails" -gt 0 ]; then
  echo
  echo "onto-skills-shell-out-check FAILED: $fails finding(s)."
  echo "Route every state mutation through the onto binary; keep schema docs in references/."
  exit 1
fi
echo "onto-skills-shell-out-check passed: onto* SKILL.md files shell out for all state writes."
```

Make it executable: `chmod +x scripts/onto-skills-shell-out-check.sh`.

- [x] **Step 2: Prove the gate FAILS before the rewrites**

This task (C1) should be committed and run **before** Layer B lands, or on a stash of the pre-B skills. Run: `./scripts/onto-skills-shell-out-check.sh`
Expected on the *pre-rewrite* skills: FAIL — it flags the no-CLI copy in `onto/SKILL.md`, the `metrics.*` references in open/design/build/verify/close/fix/tweak, and the `state.yaml` write phrasings across all eight. (If executing plan tasks strictly in order, run C1 immediately after authoring it and before/independent of the B commits to capture this failing baseline; note the finding count.)

- [x] **Step 3: Prove the gate PASSES after the rewrites (Layer B complete)**

Run: `./scripts/onto-skills-shell-out-check.sh`
Expected: PASS — "onto-skills-shell-out-check passed: …". If it still flags a line, that line is a residual hand-edit the rewrite missed — fix the skill, not the gate (unless it is the sanctioned `abandoned:` line, which the blocklist does not match).

- [x] **Step 4: Wire the check into `scripts/gate.sh`**

In `scripts/gate.sh`, add a `step` after the existing "spec<->command correspondence" step (line 46), before `govulncheck`:

```bash
step "onto skills shell out (no direct state writes)"
./scripts/onto-skills-shell-out-check.sh
```

- [x] **Step 5: Commit**

```bash
git add scripts/onto-skills-shell-out-check.sh scripts/gate.sh
git commit -m "test(onto): grep gate — onto* skills must shell out for state writes"
```

---

## Task C2: full verification gate

**Files:** none (verification only). Also update the delta spec is already present at `openspec/changes/onto-skills-shell-out/specs/onto-binary/spec.md` — confirm it validates.

- [x] **Step 1: State package + CLI, with race**

Run: `go test ./internal/ontostate/... ./internal/ontocli/... -race -count=1`
Expected: PASS (all A-task tests plus the pre-existing suite).

- [x] **Step 2: Vet and build**

Run: `go vet ./... && go build ./...`
Expected: no output, exit 0.

- [x] **Step 3: The new grep gate**

Run: `./scripts/onto-skills-shell-out-check.sh`
Expected: PASS.

- [x] **Step 4: OpenSpec validation**

Run: `openspec validate --all`
Expected: PASS — the `onto-binary` delta (new command scenarios + guides) is well-formed.

- [x] **Step 5: Full gate (optional, if a Docker daemon is available)**

Run: `./scripts/gate.sh`
Expected: ALL GATE CHECKS PASSED, including the new "onto skills shell out" step. If Docker is unavailable, run steps 1–4 above and record the Docker-E2E gap.

- [x] **Step 6: Commit any final adjustments** (only if a fix was needed; otherwise nothing to commit).

---

## Self-Review

**Spec coverage (delta `openspec/changes/onto-skills-shell-out/specs/onto-binary/spec.md`):**
- "onto new … `--workflow full|fix|tweak`, default full, reject invalid, no writes" → Task A1 (all four scenarios: chosen workflow, default full, invalid rejected, plus the unchanged clobber/name guards preserved).
- "guides accepts pending, updated, waived forms; any other rejected, no write" → Task A3 (state-model + setter tests).
- "base-ref and deps setters record creation fields" → Task A2.
- "structured read emits full state as JSON" → already implemented (`statecmd.go`); skills consume it in B1e/B1f/B1g/B2. No new work needed; confirmed present.
- "legacy state migrates on read / no schema bump" → preserved; Task A3 keeps `CurrentSchemaVersion = 1` and leaves `migrateLegacy` behavior intact (comment-only edit).

**Design-doc coverage:** Layer 1 items 1–3 → A1/A2/A3. Layer 1 item 4 (observational drop, no setters) → the metric-write removals across B1a–B1g and the migrate comment. Layer 2 (rewrite 8 skills, field→command map, delete no-CLI copy) → B1a–B1g + B2. Verification (grep gate) → C1. onto-no-slop untouched → confirmed (not in any task's file list).

**Placeholder scan:** every code/edit step shows the exact text or command. The Markdown rewrites quote the exact before/after strings and cite line ranges from the current files.

**Type consistency:** `ValidWorkflow` / `ValidGuides` exported helpers used consistently (A1, A3). `runTransition` signature matches `internal/ontocli/set.go` line 15. `enumSetterCmd` untouched. `StringArrayVar` (cobra) used for repeatable `--dep`. `fullFixtureState()` gains `Guides` before its use in round-trip and idempotency tests.

---

## Risks (flagged)

1. **Grep-gate include/exclude precision (primary).** The gate is a curated blocklist over the eight `SKILL.md` files, not a token ban — because the dispatcher legitimately *reads* the state file for discovery/derivation and `references/state-yaml.md` *documents* the schema. Both are excluded by scope (references/) or survive the blocklist (reads carry no mutation cue). The fragility: a future state-write phrased outside the blocklist's verbs would slip through. Accepted as coarse-but-real, matching `spec-command-check.sh`.

2. **Skill state writes that do NOT cleanly map to a command.** Three transitions have no binary command (all N2 / out of scope), handled by dropping the redundant *cache* write and leaning on the unchanged markdown derivation:
   - **Backward phase resets** (onto-build mid-revision `→ design`; onto-verify fail / onto reopen `→ build`): `onto advance` is forward-only. Resolved by driving the dispatcher's file-based derivation (`Status: Under revision` marker; unchecked `tasks.md`) instead of writing `phase:`.
   - **Preset phase-skip** (onto-fix/onto-tweak): `onto new` always writes `phase: open`, but presets work at build. The binary's `phase` field will read `open` for a building preset while the dispatcher derives `build`. Reconciling binary-authority with preset phase-skip is N2 — flagged in both preset skills.
   - **Workflow upgrade** (`fix|tweak → full`): no `onto set workflow`. Resolved by the proposal's `Preset: … (upgraded to full)` marker, which the dispatcher already reads as the workflow authority.

3. **Abandon path (`abandoned:` field).** No command, and `abandoned` is not in the schema — the single sanctioned direct state note, left in `onto/SKILL.md` and intentionally not matched by the gate blocklist. An `onto abandon` command is deferred to N2.

4. **Binary-vs-derivation authority tension.** This change makes the *binary* the state-write path while the *markdown dispatcher* keeps deriving phase/workflow from artifacts. For the full workflow they agree (advance is gated by the same artifacts); for presets and mid-flight revisions the binary's `phase` field lags the derived phase (risks 2). This pre-existed (change A) and is explicitly N2; the plan does not attempt to close it.
