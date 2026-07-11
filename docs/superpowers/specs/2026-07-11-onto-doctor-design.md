---
comet_change: onto-doctor
role: technical-design
canonical_spec: openspec
---

# onto doctor — Technical Design

Deep technical refinement of `openspec/changes/onto-doctor/design.md`. The
open-phase design fixed the approach (read-only, ungated, reuse existing
`ontostate` primitives, findings + non-zero exit); this document nails the
concrete signatures, finding-message formats, control flow, and test matrix.

## Command surface

```go
// doctorCmd builds the "onto doctor" subcommand: a strictly read-only,
// config-independent workspace-health diagnostic. It is NOT gated on the
// framework install (unlike init/new/close) — a missing docs layout is a
// finding, not a refusal. It writes nothing and imports none of homonto's
// projection packages.
func doctorCmd() *cobra.Command {
    var dir string
    cmd := &cobra.Command{
        Use:   "doctor",
        Short: "Report onto workflow/project health (read-only)",
        Args:  cobra.NoArgs,
        RunE:  func(cmd *cobra.Command, _ []string) error { return runDoctor(cmd, dir) },
    }
    cmd.Flags().StringVar(&dir, "dir", ".", "workspace root to inspect")
    return cmd
}
```

Registered in `root.go` between `advanceCmd()`/`closeCmd()` and the rest, so the
command list becomes advance / close / doctor / init / new / status / version.

## Control flow: `runDoctor(cmd *cobra.Command, root string) error`

Accumulate `findings []string`; append a formatted line per problem; print all
to `cmd.OutOrStdout()`; decide exit via the return value.

```
findings := nil

// 1. docs layout
for _, d := range docsLayout {                       // reuse init.go's docsLayout
    p := filepath.Join(root, d)
    info, err := os.Stat(p)
    if err != nil || !info.IsDir() {
        findings = append(findings, "docs layout: missing directory "+d)
    }
}

// 2. active changes (glob's single * excludes docs/changes/archive/<name>/…)
active, _ := filepath.Glob(filepath.Join(root,"docs","changes","*","onto-state.yaml"))
for _, path := range active {
    changeDir := filepath.Dir(path)
    name := filepath.Base(changeDir)
    st, err := ontostate.Load(path)
    if err != nil { findings = append(findings, name+": invalid onto-state.yaml: "+err); continue }
    phase, err := st.DerivePhase()
    if err != nil { findings = append(findings, name+": cannot derive phase: "+err); continue }
    if err := ontostate.ValidateSkeleton(changeDir); err != nil {
        findings = append(findings, name+": phase "+phase+" missing artifact: "+err)
    }
    if unresolved := ontostate.DepsResolved(root, st.Deps); len(unresolved) > 0 {
        findings = append(findings, fmt.Sprintf("%s: unresolved dependencies: %v", name, unresolved))
    }
    if st.Archived {
        findings = append(findings, name+": active change marked archived: true (belongs under docs/changes/archive/)")
    }
}

// 3. archive layout
entries, _ := filepath.Glob(filepath.Join(root,"docs","changes","archive","*"))
for _, entry := range entries {
    info, err := os.Stat(entry)
    if err != nil || !info.IsDir() { continue }      // archive holds change dirs; ignore stray files
    name := filepath.Base(entry)
    st, err := ontostate.Load(filepath.Join(entry, "onto-state.yaml"))
    if err != nil { findings = append(findings, "archive/"+name+": invalid or missing onto-state.yaml: "+err); continue }
    if !st.Archived { findings = append(findings, "archive/"+name+": not marked archived: true") }
}

// verdict
if len(findings) == 0 { cmd.Println("healthy"); return nil }
for _, f := range findings { cmd.Println(f) }
return fmt.Errorf("onto doctor: %d problem(s) found", len(findings))
```

(Exact string wrapping of the `err` values uses `%v`/`%w`-style formatting; the
snippet elides `fmt.Sprintf` for brevity. Findings are printed after the loop so
the summary error carries the count, and stdout holds the full list.)

### Why these primitives

| Check | Primitive | Rationale |
|-------|-----------|-----------|
| docs layout | `ontocli.docsLayout` + `os.Stat` | single source of truth with `onto init` |
| state validity | `ontostate.Load` (parse+validate) then `DerivePhase` | same load path as `onto status` |
| phase↔artifacts | `ontostate.ValidateSkeleton` | already loads+derives+checks `RequiredArtifacts(phase)` |
| deps consistency | `ontostate.DepsResolved(root, st.Deps)` | reuses #3c's resolver (archive-dir existence) |
| archived-flag consistency | `st.Archived` | an active-dir change with `archived:true` is misfiled |
| archive layout | glob `archive/*` + `Load` + `st.Archived` | mirror of the active check for the terminal set |

## Decisions locked from open-phase design

- **D1 ungated read-only** — modeled on `status.go`, not `init.go`; no `gate()`
  call; zero writes; no homonto projection imports.
- **D2 findings + exit code** — non-nil error → `main` exits 1 with `error: …`;
  `SilenceErrors/SilenceUsage` on root suppress the cobra dump.
- **D3 fixed check order** — layout → active → archive; deterministic output.
- **D4 reuse ValidateSkeleton** — accept the small double-load.

## Edge cases

- **Empty workspace** (no `docs/` at all): layout check yields 4 findings; no
  active/archive globs match; exit non-zero. Correct — an uninitialized
  workspace is "unhealthy" from onto's perspective, which is the honest report.
- **`docs/changes` empty** (no changes yet): layout passes, no change globs
  match → `healthy`. Correct — a freshly `onto init`'d workspace is healthy.
- **Stray file directly under `docs/changes/archive/`** (not a dir): ignored by
  the `os.Stat`/`IsDir` guard, not a finding.
- **Change dir without `onto-state.yaml`**: the active glob requires the state
  file, so such a dir is simply not matched (out of scope for this command;
  `onto status` behaves identically). Not a finding — consistent with status.
- **Nil/empty `st.Deps`**: `DepsResolved` returns empty (per #3c) → no finding.

## Testing strategy (TDD, RED first)

Test file `internal/ontocli/doctor_test.go`. Drive through the public root:
`cmd := NewRootCmd(); cmd.SetArgs([]string{"doctor","--dir",tmp}); cmd.SetOut(&buf)`
then `err := cmd.Execute()`. Assert on `(err == nil)` AND `buf.String()`
contents.

Seed helpers (in the test file):
- `seedDocsLayout(t, root)` — `os.MkdirAll` the four `docs/*` dirs.
- `seedActive(t, root, name, phase string, artifacts []string, deps []string)` —
  write `onto-state.yaml` (phase + deps + archived:false) and the named artifact
  files under `docs/changes/<name>/`.
- `seedArchived(t, root, name string, archived bool)` — write a change under
  `docs/changes/archive/<date>-<name>/` with `archived: <archived>`.

Cases:
1. **healthy** — full layout + one active change whose artifacts match its phase
   with resolved deps + one archived entry marked archived → `err == nil`,
   stdout == `healthy\n`.
2. **missing docs dir** — omit `docs/adr` → non-nil, stdout contains `docs/adr`.
3. **invalid active state** — write malformed YAML → non-nil, stdout names the
   change + "invalid".
4. **phase without artifact** — active at `build` without `plan.md` → non-nil,
   stdout names the missing artifact.
5. **unresolved dep** — active with `deps:[missing]`, no archived `missing` →
   non-nil, stdout contains `missing`.
6. **active marked archived** — active change state `archived:true` → non-nil,
   stdout contains `archived`.
7. **malformed archive entry** — archive dir with state `archived:false` →
   non-nil, stdout contains the archive entry name.
8. **ungated read-only** — no `homonto.toml`, missing docs layout → still runs
   (non-nil for layout findings) AND assert no new files were created
   (snapshot the root's file set before/after, or assert `homonto.toml` and
   `docs/` remain absent).

Isolation guard in Task 1.3: `grep -nE "internal/(config|engine|adapter|catalog)"
internal/ontocli/*.go` stays empty.

## Non-goals (restated)

No `--fix`, no installation/projection checks (that's `homonto doctor`), no
configurable artifact roots, no state-content checks beyond the existing
validators.
