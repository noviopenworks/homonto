---
change: homonto-v1-core
design-doc: docs/superpowers/specs/2026-07-03-homonto-v1-core-design.md
base-ref: 83785cef924e1aa9b0e17c01a7e276714b81ca1a
---

# homonto v1 core — Build Plan

Executes `openspec/changes/homonto-v1-core/tasks.md` (21 items). Mechanical
step-by-step code for the unmarked items is the canonical **base plan**
`docs/superpowers/plans/2026-06-24-homonto.md` (Tasks 1–14). This file records the
**⚑ hashed-state overrides** that supersede the base plan, per the Design Doc.

## Global constraints

- Module `github.com/noviopenworks/homonto`, Go floor 1.22 (toolchain 1.26 present).
- TDD (Red→Green→Refactor): failing test first for every unit; commit per task.
- Secrets referenced, resolved after confirm, all-at-once before any write.
- Surgical merge; atomic temp+rename; `state.json` written last.
- **Plan output and `state.json` never contain a resolved secret.**
- DRY, YAGNI. One commit per completed task, message reflects intent.

## ⚑ Hashed-state secret-idempotency overrides (supersede base plan)

These replace the base plan's naive `st.Set(tool, key, c.New)` state model.

### O-1 — `state.Entry` schema (supersedes base Task 4)
`internal/state/state.go`:
```go
type Entry struct {
    Desired string `json:"desired"` // unresolved value, may contain ${...}
    Applied string `json:"applied"` // sha256(resolved value written to disk)
}
type State struct {
    Managed map[string]map[string]Entry `json:"managed"`
}
func (s *State) Set(tool, key, desired, appliedHash string)
func (s *State) Get(tool, key string) (Entry, bool)
```
`Load`/`Save` unchanged in behavior (absent→empty, atomic write). Tests updated
to the `Entry` round-trip.

### O-2 — `Hash` helper
Add `func Hash(s string) string` returning lowercase hex `sha256` (in
`internal/secret`, reused by adapters/state). Test: stable + differs by input.

### O-3 — Adapter `Plan` decision (supersedes base Tasks 8, 9 plan logic)
For each desired managed key, with `want = desired (unresolved)`,
`disk = on-disk value or absent`, `e, inState = st.Get(tool, key)`:
```
disk absent                                   → create (New: want; Old: "")
!secret.ContainsRef(want):
    jsonEqual(disk, want) ? noop : update (Old: disk,        New: want)
secret.ContainsRef(want):
    inState && e.Desired == want && e.Applied == secret.Hash(disk)
        ? noop
        : update (Old: "«secret»",  New: want)   // NEVER put disk in Old
```
`create` on a secret key also uses `Old: ""` (no disk value). Normalize both
sides of every non-secret compare through JSON marshal/unmarshal (`jsonEqual`).

### O-4 — Adapter `Apply` records hashed state (supersedes base Tasks 8, 9)
After resolving a non-noop change to `resolved` and writing it:
```go
st.Set(tool, c.Key, c.New, secret.Hash(resolved))
```
Store the unresolved `c.New` as `Desired`, the hash of the resolved value as
`Applied`. Never store `resolved` itself.

### O-5 — Secret-safety test extension (supersedes/extends base Task 6/8)
Add a test: a secret-backed key whose on-disk resolved value has drifted →
`Plan` yields an `update` whose rendered output (and `Change.Old`) contains
neither the resolved value nor anything but `«secret»`; and after any apply,
`state.json` bytes contain no resolved secret.

### O-6 — Drift uses the same logic (supersedes base Task 11)
`Engine.Drift` reports keys where `Plan` yields `update` and state has an entry.
Secret-key drift is detected via `Applied != Hash(disk)` and reported without
printing the value.

### O-7 — e2e idempotency includes a secret (supersedes base Task 14)
The end-to-end test's config includes a secret-backed MCP env; assert the second
`Plan` after `Apply` has no real changes (`plan.HasChanges == false`).

## Task list (mirrors OpenSpec tasks.md)

Follow the base plan for each item's Red/Green steps EXCEPT where an ⚑ override
above applies. Check off the item in both this plan and
`openspec/changes/homonto-v1-core/tasks.md` on completion, then commit.

1. [x] **1.1** Scaffold module + `version` (base Task 1)
2. [x] **1.2** Config model + TOML loader (base Task 2)
3. [x] **1.3** Secret resolver (base Task 3)
4. [x] **1.4** ⚑ `Hash` helper — O-2
5. [x] **2.1** ⚑ State store with `Entry{Desired,Applied}` — O-1
6. [x] **2.2** Surgical JSON/JSONC merge (base Task 5)
7. [x] **2.3** Content linker (base Task 7)
8. [x] **3.1** Adapter interface + `Change`/`ChangeSet` + plan printer (base Task 6)
9. [x] **3.2** ⚑ Claude adapter — base Task 8 + O-3, O-4 (redact `Old`; hashed state)
10. **3.3** ⚑ OpenCode adapter + Claude skill linking — base Task 9 + O-3, O-4
11. **3.4** ⚑ Secret-safety tests incl. drift path — O-5
12. **4.1** Engine + `plan`/`apply` (base Task 10)
13. **4.2** `status` (drift ⚑ O-6) + `doctor` (base Task 11)
14. **4.3** `init` scaffold (base Task 12)
15. **4.4** `import` with secret redaction (base Task 13)
16. **5.1** ⚑ e2e incl. secret-backed idempotency — base Task 14 + O-7
17. **5.2** Two-phase abort test (base Task 10 test)
18. **5.3** Golden-file surgical-merge tests (fold into 3.2/3.3 where already covered)
19. **5.4** README (base Task 14 Step 3)
20. **5.5** Full suite green: `go test ./... && go vet ./... && go build ./...`

Note: base-plan Tasks 8/9 tests that assert secret-backed idempotency must be
written against the ⚑ hashed-state model (state seeded so a repeat plan is a
noop), not the base plan's disk-vs-unresolved comparison.
```
