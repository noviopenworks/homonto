# Onto Framework Harsh Review

**Reviewed:** 2026-07-13
**Scope:** Onto workflow, Homonto projection engine, Comet/OpenSpec integration,
security, extensibility, CLI, documentation, tests, CI, and releases.

## Verdict

Onto is not currently a state-of-the-art spec-driven framework. It is two
incompatible workflows sharing one name:

- a sophisticated prose-driven skill framework using `state.yaml`; and
- a much weaker Go workflow engine using `onto-state.yaml`.

The underlying Homonto projector is substantially stronger than the workflow
framework around it. The codebase has excellent safety instincts, tests, ADRs,
and remote-content controls, but the spec-driven claim is undermined by stale
canonical specs, non-semantic gates, fragmented state, and documentation drift.

### Subjective scorecard

| Area | Score |
|---|---:|
| Core projection engine | 8/10 |
| Test investment | 8/10 |
| Security foundations | 7/10 |
| Onto workflow coherence | 2/10 |
| Spec truth and traceability | 3/10 |
| Extensibility | 4/10 |
| CLI/automation experience | 5/10 |
| State-of-the-art readiness | **4/10** |

## Critical blockers

### 1. Onto has two incompatible control planes

The skill framework says Onto is markdown-only, has no external CLI, and stores
state in `docs/changes/<name>/state.yaml`
(`catalog/skills/onto/SKILL.md:8-14`). The binary instead owns
`onto-state.yaml` (`internal/ontostate/state.go:26-35`).

`onto-open` creates `state.yaml` (`catalog/skills/onto-open/SKILL.md:57-73`),
while `onto advance` and `onto close` only read `onto-state.yaml`
(`internal/ontocli/advance.go:64-69`, `internal/ontocli/close.go:50-55`).

**Result:** each side can report a different phase, workflow, and archive state.
This must be resolved before adding features.

### 2. The binary can archive empty, unverified work

The binary checks artifact existence, not artifact validity:

- Required artifacts are just filenames
  (`internal/ontostate/state.go:125-136`).
- Advancing checks files and task checkboxes
  (`internal/ontocli/advance.go:77-109`).
- Closing checks phase, dependencies, and worktree cleanliness
  (`internal/ontocli/close.go:58-90`).

It does not enforce:

- confirmed design;
- scenario coverage;
- `Result: pass`;
- current verification evidence;
- merged delta specs;
- ADR resolution; or
- guide resolution.

A user can create empty files, check tasks, advance repeatedly, and archive.

### 3. Binary state serialization destroys workflow metadata

The binary state contains only seven fields
(`internal/ontostate/state.go:26-35`). The skill schema includes decisions,
directives, verification result, close progress, guides, metrics, and
abandonment (`catalog/skills/onto/references/state-yaml.md:12-44`).

Because the binary marshals only its typed struct
(`internal/ontostate/state.go:88-113`), richer state would be dropped on save.

### 4. Close is interruption-unsafe

The binary writes `archived: true` before creating or moving the archive
(`internal/ontocli/close.go:80-89`). A failed rename leaves an active workspace
marked archived and potentially undiscoverable.

The skill workflow similarly uses coarse `close.merged` state for a
multi-operation merge (`catalog/skills/onto/references/state-yaml.md:29-32`). A
crash halfway through can cause recovery to skip unfinished operations.

**Required fix:** journal individual close operations and commit completion only
after post-operation validation.

### 5. Canonical specifications contain removed behavior

The canonical agent lifecycle still requires an imperative `homonto agents`
subsystem (`openspec/specs/agent-lifecycle/spec.md:3-11`). The CLI command spec
also requires it (`openspec/specs/cli-commands/spec.md:9-21`), but the current
CLI no longer registers that subsystem.

The validator still reports all 15 specs valid. Validation therefore proves
syntax, not truth.

### 6. Remote digest changes are invisible and bypass confirmation

`homonto plan` can report no changes for a digest-only remote repin. `apply`
then performs remote mutation without confirmation
(`internal/cli/apply.go:47-60`).

This directly violates the plan/confirm/apply principle: instructions can
change after the user is told there are no projection changes.

### 7. Tampered state can authorize arbitrary file deletion

Projection state accepts unvalidated `Entry.Desired` paths
(`internal/state/state.go:41-57`). Copy-mode adapters treat those paths as
destinations (`internal/adapter/claude/claude.go:248-272`,
`internal/adapter/opencode/opencode.go:244-266`).

If a recorded arbitrary file matches the recorded hash, copy pruning removes it
(`internal/copyfile/copyfile.go:103-115`,
`internal/copyfile/copyfile.go:144-146`).

State paths must be reconstructed from validated resource identities, never
trusted as absolute authority.

### 8. Remote application mutates before all remotes verify

Remote processing prunes old content first, then verifies and materializes
resources sequentially (`internal/engine/remote.go:62-103`).

If a later remote fails verification:

- earlier content may already have changed;
- old content may already be pruned; and
- the lockfile may remain stale.

All remotes must verify into staging before any active content or lock mutation.

## High-severity workflow and specification findings

### 9. Fix and tweak workflows are incompatible with binary gates

The skill presets intentionally skip design or plan artifacts, while the binary
always requires them in build and verify
(`internal/ontostate/state.go:125-133`). The binary also hardcodes full workflow
creation.

Add workflow-aware transitions and `onto new --workflow full|fix|tweak`.

### 10. Dependency semantics disagree

The skill requires date-anchored exact archive names and says an active
workspace overrides an archive (`catalog/skills/onto/SKILL.md:69-82`).

The binary uses a loose `*-<dep>` glob and enforces dependencies only at close
(`internal/ontostate/state.go:189-198`,
`internal/ontocli/close.go:62-65`).

Dependencies should block entering build, detect cycles, and use one shared
resolver.

### 11. Verification is not genuinely spec-gated

Comet's lightweight verification can skip scenario coverage and drift analysis.
Scale selection is based on task and file counts rather than risk or changed
requirements (`catalog/skills/comet/scripts/comet-runtime.mjs:12959-12986`).

A one-file security change can receive less scrutiny than a harmless large
refactor. Any changed delta spec should force requirement-level evidence.

### 12. Verification escape hatches are too broad

Full verification can continue without independent reviewers, and failed
findings can be accepted too generally. Security, data-loss, and failed core
acceptance criteria need non-waivable classes.

The Onto framework also does not bundle the skeptic or reviewer subagents
required by its ideal process.

### 13. Requirement traceability is prose-based

Requirements and scenarios have mutable headings instead of stable IDs.
Verification reports manually map names to tests.

The toolkit cannot reliably answer: which code, tests, decisions, commits, and
release prove scenario X?

Introduce immutable IDs and typed relationships such as `implements`, `tests`,
`supersedes`, `deviates-from`, and `released-in`.

### 14. Missing state is invisible to recovery diagnostics

Status and doctor enumerate directories containing `onto-state.yaml`, so a real
workspace whose state file was deleted can disappear from diagnostics entirely.

Enumerate change directories first, then classify them as valid, malformed, or
missing-state.

### 15. Isolation is selected too late

Onto commits open and design work before selecting a branch or worktree during
build. This can put planning commits directly on `main`.

Isolation should be selected before the first workspace commit.

### 16. Recovery permits dangerous cleanup assumptions

Current build recovery may stash or reset dirty work based on the assumption
that it belongs to an interrupted task. It cannot safely distinguish user work,
concurrent-agent work, and failed-task output.

Recovery should preserve dirty paths into a patch or WIP branch and require
explicit authorization before destructive reset.

### 17. Parallel task bookkeeping is not atomic

Workers commit task code, then the coordinator separately merges branches and
records task checkoffs. A crash between those steps causes completed work to
appear incomplete and risks redispatch.

Integrate code and checkoff into one coordinator-owned task commit.

### 18. Archive metadata is systematically broken

Archived `.comet.yaml`, handoff, and artifact files retain paths into removed
active directories. Archive is therefore where long-term traceability breaks.

Archive must:

1. move all historical artifacts;
2. rewrite references;
3. validate every referenced path and hash; and
4. only then mark the change archived.

### 19. Active-work documentation contains completed history

`docs/superpowers/README.md` declares the directory active-only, but completed
designs and a mostly unchecked historical plan remain there. These contradict
the roadmap's completed status.

### 20. Public capability claims conflict

README, user guides, release notes, roadmap, and specs disagree about:

- Codex support;
- remote resources;
- available adapters;
- agent lifecycle;
- output streams; and
- supported targets.

Release notes still publish obsolete limitations.

### 21. Onto, Comet, and Homonto lack a clear product hierarchy

The repository presents:

- Homonto as projection and distribution;
- Onto as a self-contained workflow;
- Comet as another full workflow; and
- OpenSpec and Superpowers as additional governance layers.

There is no strong persona or selection matrix explaining which workflow users
should choose or how artifacts migrate between them.

### 22. The roadmap is a completion diary, not strategy

It labels a twelve-month horizon delivered while major foundational
contradictions remain. Deferred provenance work effectively disappeared.

Separate release history from forward product strategy.

### 23. Comet's mandatory build guard does not support Go

The bundled runtime recognizes npm, Maven, and Cargo but not Go, despite this
repository being Go. The bypass is undocumented.

A framework dogfooding itself should not require a hidden skip flag for its own
language.

### 24. Generated Comet runtime lacks reproducible provenance

The repository ships a roughly 13,000-line generated bundle without:

- authored sources;
- pinned upstream revision;
- generator version;
- generated-file header; or
- regeneration-diff CI check.

Embedded YAML is duplicated inside the bundle. This is an unreviewable
supply-chain surface.

## High-severity security and transactional findings

### 25. Atomic writer follows arbitrary symlinks

`fsutil.WriteAtomic` resolves symlinks and writes through them. This is unsafe
for `.homonto` state, cache, lock, catalog, and remote materialization paths.

Control-plane files need no-follow path traversal and root confinement.

### 26. Cache race can accept unverified winner content

If cache rename loses a race, `Cache.Put` accepts whatever directory now exists
without rehashing it (`internal/remote/cache.go:52-56`).

Reverify the winning directory before acceptance and use per-digest locking.

### 27. Git limits are enforced after expensive fetch and checkout

Size and file-count limits occur after Git downloads and checks out content. Git
also receives an unbounded context.

A malicious pinned repository can exhaust time, bandwidth, or disk before
validation.

### 28. `local:` skill and command sources permit traversal

Local source suffixes are joined directly into provider paths
(`internal/adapter/claude/claude.go:313-320`,
`internal/adapter/opencode/opencode.go:306-314`).

Subagents receive stronger plain-name validation than skills and commands.
Apply the same validation and verify the cleaned path remains under the provider
root.

### 29. No cross-process locking or compare-and-swap

Two concurrent applies can plan from the same snapshot and last-writer-win state
and tool files. Apply and cache GC can also race.

Introduce project-scoped locking plus generation or fingerprint checks after
confirmation.

### 30. Revocation and remote drift do not deactivate content

Revoked or modified materialized remote content may remain linked and active
after apply fails. Status mostly observes link targets rather than verified
target bytes.

Doctor and status should verify materialized digests and quarantine revoked
content.

### 31. Credential-bearing locators may leak

Remote URL errors and lock entries retain full locators. Embedded userinfo or
query tokens can appear in logs and files.

Reject credentials in locators or use separate secret-backed headers and
redacted canonical URLs.

### 32. Release publication uses mutable executable dependencies

Release workflows use mutable action tags and `govulncheck@latest`. Checksums
are unsigned and created by the same potentially compromised job.

Pin action SHAs and tool versions; produce signed provenance, SBOMs, and
attestations.

## Extensibility and architecture findings

### 33. The adapter contract is not a real plugin interface

The interface depends directly on concrete config, mutable global state, and
secret resolver types. Engine construction hardcodes all adapters.

A fourth adapter requires central edits across config, engine, validation,
status, doctor, and tests.

### 34. Configuration is closed over current tools

Models, plugins, TUI, marketplaces, and target validation directly encode
Claude, OpenCode, and Codex. Adding a tool has repository-wide blast radius.

Move to capability descriptors keyed by `ToolID`.

### 35. Local frameworks can validate and silently install nothing

The generic source validator permits `local:` frameworks, but framework
expansion only acts on `builtin:` sources.

Unsupported combinations must fail at load; declarations must never become
silent no-ops.

### 36. Framework composition is a bundle installer, not an ecosystem model

Frameworks lack:

- compatibility constraints;
- required and provided capabilities;
- optional features;
- controlled overrides;
- migrations;
- conflict policy;
- local or custom framework resolution; and
- conformance contracts.

This is opinionated, but not flexible.

### 37. Versioning fields are largely decorative

Agent and subagent versions are informational; framework versions do not drive
dependency resolution; state has no schema version; compatibility migration is
ad hoc.

Add explicit config and state schema versions with ordered, idempotent
migrations.

### 38. Plugin is only native tool configuration

There is no plugin lifecycle, discovery, registration, compatibility, or
version model. Rename it to `ToolPluginDeclaration` or introduce a true
adapter-owned plugin capability.

### 39. Codex support is inconsistent

Codex supports only opt-in MCP projection, while documentation and validation
imply broader or conflicting support. Doctor does not provide equivalent
coverage.

Generate target validation and documentation from an executable capability
matrix.

### 40. Claude and OpenCode adapters duplicate security-sensitive logic

Both are nearly 1,000 lines and duplicate planning, linking, pruning, copy mode,
adoption, and drift behavior.

Codex's small `structproj`-based design is the better architectural direction.

### 41. Plans are stringly typed and not self-contained

Plan actions and payloads are unrestricted strings. `Apply` relies on mutable
adapter fields populated by a prior `Plan` call rather than solely on the
supplied plan.

Introduce typed immutable operations and reject unknown actions and tools.

### 42. Adapter application is not transactional

Adapters write several files and links sequentially. A late failure can leave
live filesystem changes without matching persisted state.

At minimum, add operation journals and failure-injection coverage at every
boundary.

### 43. Config loading is a monolith

One large file handles decoding, legacy migration, normalization, defaults,
validation, framework expansion, and tool policy. Similar resource expansion is
repeated by kind.

Split it into explicit phases with a generic expansion pipeline.

### 44. Error semantics hide incomplete assessments

Adapter planning failures become mutable warning side channels. Observation
often treats unreadable files like absent files.

Use typed results distinguishing clean, changed, degraded, unreadable, and
failed.

## CLI and operational UX findings

### 45. Plan and status can claim success after skipping adapters

A failed adapter may produce a warning followed by `Everything up to date` or
`No drift`, while exiting zero.

Never print clean conclusions when coverage was incomplete. Add detailed exit
codes.

### 46. Catalog upgrades can be skipped

When no projection changes exist, CLI apply may return before catalog
reconciliation. Doctor does not reliably report catalog-version mismatch.

Catalog updates must appear in the plan.

### 47. Catalog materialization is destructive and non-atomic

Current catalog directories are removed before replacements are complete.
Existing symlinks can temporarily or permanently point to missing content after
failure.

Build a versioned staging tree and atomically swap it.

### 48. `import --force` can destroy valid configuration

A malformed source tool config may become an empty import plus warning;
`--force` can then overwrite a valid Homonto config.

Source parse failure must be fatal before destination mutation. Import writes
should be atomic and backed up.

### 49. Most commands silently accept positional arguments

A user can run `homonto apply production.toml`, expecting that file to be used,
while the positional argument is ignored.

Use `cobra.NoArgs` consistently and show `--config` examples.

### 50. No stable automation contract exists

There is no versioned JSON output and no useful exit-code taxonomy for:

- drift;
- pending changes;
- degraded scans;
- warnings;
- user abort; or
- doctor findings.

Add `--output text|json` and detailed exit statuses.

### 51. Remote maintenance is incomplete

Cache GC exists internally but has no CLI command. Git operations lack bounded
execution, and multi-remote progress is not surfaced.

### 52. Init handles existing repositories poorly

Existing files are silently skipped. Existing `.gitignore` files are not
augmented or clearly warned about. New target directories are not handled
ergonomically.

## Testing, CI, and release findings

### 53. CI does not run the real-tool E2E matrix

Mandatory gates run stub or file-oriented Docker tests, while the real Claude
and OpenCode harness is separate. Some E2E assertions discard real command exit
status with `|| true`.

### 54. Cross-platform support is mostly cross-compilation

CI runs on Ubuntu. macOS and Windows binaries are built but not natively
exercised, despite symlink-heavy behavior being platform-sensitive.

### 55. Adapter conformance is incomplete

There is no reusable contract suite covering every adapter's:

- create, update, and delete;
- adoption;
- drift;
- unmanaged preservation;
- secret redaction;
- malformed documents;
- conflict safety; and
- byte stability.

### 56. Coverage is strong but unmanaged

The suite has high aggregate coverage, but CI does not publish or enforce a
non-regression threshold. Important observation and doctor paths remain weakly
covered.

### 57. Fuzzing is one-off rather than continuous

Useful fuzz targets exist, but CI runs only seed cases. Add scheduled fuzz
campaigns and preserve discovered corpus cases.

### 58. No performance benchmarks exist

Remote extraction permits large trees, while config expansion, plan and apply,
merge, and archive operations have no benchmark or allocation budgets.

### 59. Toolchain compatibility is unclear

Documentation says Go 1.23+, while `go.mod` requires a newer toolchain. Real
AI-tool installers and security tools are mutable and unpinned.

### 60. Release policy does not enforce semantic or provenance constraints

Any `v*` tag can trigger publication. SemVer, protected ancestry, approval,
signatures, and attestations are not enforced.

### 61. Historical archive material creates substantial repository noise

Thousands of lines of generated `.comet` recovery and handoff state are
retained. This hurts indexing, search, and audit clarity.

Design a thin archive format retaining canonical artifacts, hashes, and
necessary evidence rather than complete runtime residue.

## Strengths worth preserving

- Plan, confirm, apply, and secret non-resolution are well designed.
- Drift is correctly separated from unapplied desired-state changes.
- State is persisted after each successful adapter, supporting convergent
  partial recovery.
- Remote pinning, revocation, extraction limits, and traversal protection are
  unusually serious.
- Catalog dependency traversal is deterministic and cycle-aware.
- Link and copy ownership is conservative about foreign content.
- ADRs honestly record reversals and trade-offs.
- Human gates, notes, and adversarial verification are good ideas when
  deterministically enforced.
- The test suite is substantial: 524 tests across 29 packages, including race
  and fuzz coverage.

## Recommended path to state of the art

### P0: Make it truthful and safe

1. Choose one Onto authority, preferably the binary plus one versioned state
   schema.
2. Migrate skills to invoke deterministic transitions instead of directly
   editing state.
3. Make semantic verification and close evidence mandatory.
4. Correct stale canonical specs immediately.
5. Fix arbitrary state-path deletion and local-source traversal.
6. Stage and verify all remotes before mutation.
7. Add no-follow filesystem policy and project locking.

### P1: Build the real spec-driven core

1. Stable IDs for capabilities, requirements, scenarios, decisions, tasks, and
   evidence.
2. A typed traceability graph validated in CI.
3. Immutable typed plans.
4. Transaction journals for apply and close.
5. Workflow profiles: full, fix, tweak, and custom policy overlays.
6. A capability registry for adapters and source providers.
7. Explicit config and state migrations.

### P2: Become ecosystem-grade

1. Versioned framework manifests with dependencies, capabilities, and
   compatibility ranges.
2. Local and custom framework support.
3. Machine-readable output and detailed exit codes.
4. A reusable adapter conformance suite.
5. Signed catalogs, SBOMs, and release provenance.
6. Native macOS and Windows CI plus pinned real-tool E2E.
7. Generated documentation and capability matrices from executable metadata.

## Verification evidence

The review used fresh verification:

- `go test ./... -count=1` — 524 passed.
- `go test -race ./... -count=1` — 524 passed.
- `go vet ./...` — no issues.
- `go build ./...` — success.
- `openspec validate --all --no-color` — 15/15 syntactically valid.
- `git diff --check` — clean.

The fact that OpenSpec reports 15/15 while canonical specs describe removed
behavior is itself a major finding: the current validator checks form, not
correspondence to reality.

Graphify was not updated. At review time, the only pre-existing worktree changes
were:

- `catalog/skills/onto-build/references/subagent-protocol.md`
- `catalog/skills/onto-close/SKILL.md`
