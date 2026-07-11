# Agentic Workflows Roadmap Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the release-integrity gap by fixing agent ownership defects, aligning living documentation with source, and adding compiled-binary Docker and packaging evidence for the implemented Homonto and Onto workflows.

**Architecture:** Preserve the existing Go command and adapter architecture. Fix ownership invariants at the lifecycle boundary, then extend the current Docker harness into independent suites orchestrated by one entrypoint. Keep release verification in repository scripts so local runs, CI, and tag publication execute the same gate.

**Tech Stack:** Go 1.23, Cobra, POSIX shell, Bash, Docker, GitHub Actions, OpenSpec Markdown specifications.

## Global Constraints

- Do not add remote source support, a new adapter, or an interactive TUI in this plan.
- Never drop an agent lock record while a managed install or conflict sidecar still requires cleanup.
- Never overwrite or remove user-owned content without the existing backup/conflict policy.
- Docker scenarios must execute compiled `homonto` and `onto` binaries against disposable `HOME` and workspace directories.
- File/state/archive assertions are preferred over stdout matching; use stdout only for explicit CLI output contracts.
- Every behavior change starts with a failing focused test and ends with the full Go and Docker gates.
- Keep Docker suites independently runnable and independently reported.
- Do not refactor the adapters or agent CLI beyond what these safety fixes and test seams require.

## File Structure

- Modify `internal/cli/agents.go`: preserve de-declared target ownership and handle prune deletion errors.
- Modify `internal/cli/agents_update_test.go`: regression for update after target removal.
- Modify `internal/cli/agents_prune_test.go`: regression for failed install and sidecar deletion.
- Modify `test/docker/Dockerfile`: build both binaries and install required shell/git tools only if the base image lacks them.
- Rename `test/docker/smoke.sh` to `test/docker/homonto-core.sh`: preserve the current passing core smoke unchanged except for suite labeling.
- Create `test/docker/smoke.sh`: suite orchestrator.
- Create `test/docker/homonto-expanded.sh`: framework, catalog, plugin, marketplace, and TUI projection.
- Create `test/docker/homonto-agents.sh`: compiled agent lifecycle coverage.
- Create `test/docker/onto-lifecycle.sh`: compiled Onto lifecycle and gate coverage.
- Create `scripts/test-release-artifacts.sh`: deterministic dual-binary archive assertions.
- Create `scripts/verify-release.sh`: shared full release gate.
- Modify `.github/workflows/ci.yml`: run named Docker suites and packaging verification.
- Modify `.github/workflows/release.yml`: block publication on the shared release gate.
- Modify `README.md`, `docs/road-to-release.md`, `docs/release-checklist.md`, `docs/release-notes.md`, and living specs under `openspec/specs/`: align claims with source and evidence.

---

### Task 1: Preserve De-declared Agent Target Ownership

**Files:**
- Modify: `internal/cli/agents_update_test.go`
- Modify: `internal/cli/agents.go:585-697`

**Interfaces:**
- Consumes: `agentlock.Agent.Installed map[string]agentlock.Install`
- Produces: `runAgentUpdate` preserves records for targets absent from the current declaration so `agents doctor` and `agents prune` can still find them.

- [ ] **Step 1: Add the failing target-removal regression**

Append this test to `internal/cli/agents_update_test.go`:

```go
func TestAgentsUpdatePreservesDeDeclaredTargetForPrune(t *testing.T) {
	home := t.TempDir()
	cfg, cfgDir := addWorkspace(t, copyAgentTOML, map[string]string{"rev": "# rev\n"})
	if out, err := runCmd(t, home, "", "agents", "add", "rev", "--config", cfg); err != nil {
		t.Fatalf("agents add: %v\n%s", err, out)
	}

	claudeOnly := "[agents.rev]\nsource=\"local:rev\"\nmode=\"copy\"\ntargets=[\"claude\"]\n"
	if err := os.WriteFile(cfg, []byte(claudeOnly), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := runCmd(t, home, "", "agents", "update", "rev", "--config", cfg); err != nil {
		t.Fatalf("agents update: %v\n%s", err, out)
	}

	lock, err := agentlock.Load(filepath.Join(cfgDir, ".homonto"))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := lock.Agents["rev"].Installed["opencode"]; !ok {
		t.Fatal("update dropped the de-declared target record before prune")
	}
	if _, err := os.Lstat(opencodeDst(home, "rev")); err != nil {
		t.Fatalf("de-declared target must remain installed until prune: %v", err)
	}

	if out, err := runCmd(t, home, "", "agents", "prune", "--config", cfg); err != nil {
		t.Fatalf("agents prune: %v\n%s", err, out)
	}
	if _, err := os.Lstat(opencodeDst(home, "rev")); !os.IsNotExist(err) {
		t.Fatalf("prune must remove the de-declared target, got %v", err)
	}
}
```

- [ ] **Step 2: Run the focused test and confirm the current defect**

Run: `go test ./internal/cli -run TestAgentsUpdatePreservesDeDeclaredTargetForPrune -count=1`

Expected: FAIL with `update dropped the de-declared target record before prune`.

- [ ] **Step 3: Preserve existing install records before reconciling active targets**

Replace the empty `installedRec` initialization in `runAgentUpdate` with:

```go
	installedRec := make(map[string]agentlock.Install, len(inst.Installed))
	for tool, install := range inst.Installed {
		installedRec[tool] = install
	}
```

The existing target loop continues to replace records for currently declared
targets. Records for removed targets remain available to doctor and prune.

- [ ] **Step 4: Run focused lifecycle tests**

Run: `go test ./internal/cli -run 'TestAgents(Update|Prune)' -count=1`

Expected: PASS.

- [ ] **Step 5: Commit the invariant fix**

```bash
git add internal/cli/agents.go internal/cli/agents_update_test.go
git commit -m "fix(agents): preserve removed targets until prune"
```

### Task 2: Retain Ownership When Prune Deletion Fails

**Files:**
- Modify: `internal/cli/agents_prune_test.go`
- Modify: `internal/cli/agents.go:65-94`

**Interfaces:**
- Consumes: recorded `agentlock.Install.Path` and optional `<path>.merged`.
- Produces: prune drops a record only after both the install and conflict sidecar are absent or successfully removed.

- [ ] **Step 1: Add a failing install-deletion regression**

Append:

```go
func TestAgentsPruneRemoveFailureKeepsRecord(t *testing.T) {
	home := t.TempDir()
	toml := "[agents.rev]\nsource=\"local:rev\"\nmode=\"copy\"\ntargets=[\"claude\"]\n"
	cfg, cfgDir := addWorkspace(t, toml, map[string]string{"rev": "# rev\n"})
	if out, err := runCmd(t, home, "", "agents", "add", "rev", "--config", cfg); err != nil {
		t.Fatalf("agents add: %v\n%s", err, out)
	}

	dst := claudeDst(home, "rev")
	if err := os.Remove(dst); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dst, "keep"), []byte("user data"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte("\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCmd(t, home, "", "agents", "prune", "--config", cfg)
	if err != nil {
		t.Fatalf("prune should isolate the target failure: %v\n%s", err, out)
	}
	if !strings.Contains(out, "SKIPPED") || !strings.Contains(out, "remove failed") {
		t.Fatalf("prune must report deletion failure, got:\n%s", out)
	}
	lock, loadErr := agentlock.Load(filepath.Join(cfgDir, ".homonto"))
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	if _, ok := lock.Agents["rev"]; !ok {
		t.Fatal("failed deletion must retain the lock record")
	}
}
```

- [ ] **Step 2: Run the regression and confirm it fails**

Run: `go test ./internal/cli -run TestAgentsPruneRemoveFailureKeepsRecord -count=1`

Expected: FAIL because current prune reports removal and drops the record.

- [ ] **Step 3: Check every removal and make missing-primary retries clean sidecars**

Add `errors` to the imports in `internal/cli/agents.go`, then replace the
unconditional removals in `pruneFile` with this control flow:

```go
				removeSidecar := func(path string) bool {
					if err := os.Remove(path + ".merged"); err != nil && !errors.Is(err, os.ErrNotExist) {
						actions = append(actions, fmt.Sprintf("SKIPPED %s.merged: remove failed (%v); record kept", path, err))
						return false
					}
					return true
				}

				pruneFile := func(ti agentlock.Install) bool {
					if _, err := os.Lstat(ti.Path); err != nil {
						if errors.Is(err, os.ErrNotExist) {
							return removeSidecar(ti.Path)
						}
						actions = append(actions, fmt.Sprintf("SKIPPED %s: inspect failed (%v); record kept", ti.Path, err))
						return false
					}
					if dryRun {
						actions = append(actions, fmt.Sprintf("would remove %s", ti.Path))
						return true
					}
					if b, rerr := os.ReadFile(ti.Path); rerr == nil && agentlock.HashContent(b) != ti.Hash {
						if err := fsutil.WriteAtomic(ti.Path+".bak", b); err != nil {
							actions = append(actions, fmt.Sprintf("SKIPPED %s: backup to .bak failed (%v); file kept", ti.Path, err))
							return false
						}
						actions = append(actions, fmt.Sprintf("backed up %s to %s.bak", ti.Path, ti.Path))
					}
					if err := os.Remove(ti.Path); err != nil {
						actions = append(actions, fmt.Sprintf("SKIPPED %s: remove failed (%v); record kept", ti.Path, err))
						return false
					}
					if !removeSidecar(ti.Path) {
						return false
					}
					actions = append(actions, fmt.Sprintf("removed %s", ti.Path))
					return true
				}
```

- [ ] **Step 4: Run all prune tests**

Run: `go test ./internal/cli -run TestAgentsPrune -count=1`

Expected: PASS.

- [ ] **Step 5: Commit the deletion-safety fix**

```bash
git add internal/cli/agents.go internal/cli/agents_prune_test.go
git commit -m "fix(agents): retain ownership when prune fails"
```

### Task 3: Align Agent CLI Contracts and Living Specifications

**Files:**
- Modify: `internal/cli/agents.go:167-215,353-357,476-481`
- Modify: `internal/cli/agents_builtin_test.go`
- Modify: `openspec/specs/agent-lifecycle/spec.md`
- Modify: `openspec/specs/subagent-projection/spec.md`
- Modify: `openspec/specs/framework-expansion/spec.md`
- Modify: `openspec/specs/builtin-catalog/spec.md`
- Modify: `openspec/specs/command-projection/spec.md`
- Modify: `openspec/specs/onto-binary/spec.md`
- Modify: `openspec/specs/tool-adapters/spec.md:24-35`

**Interfaces:**
- Produces: help and remediation text accurately names local and builtin sources; every living spec has a concrete purpose and current behavior.

- [ ] **Step 1: Add CLI contract assertions**

Add this test to `internal/cli/agents_builtin_test.go`:

```go
func TestAgentsHelpDescribesAllSupportedSources(t *testing.T) {
	if got := agentsAddCmd().Short; got != "Install a declared agent and record it" {
		t.Fatalf("agents add short = %q", got)
	}
	if got := agentsUpdateCmd().Short; got != "Reconcile installed agents with their declared sources" {
		t.Fatalf("agents update short = %q", got)
	}
}
```

In the existing source-change doctor test in
`internal/cli/agents_doctor_test.go`, add:

```go
	if !strings.Contains(out, "homonto agents update rev") {
		t.Fatalf("source-change remediation must point to agents update, got:\n%s", out)
	}
```

- [ ] **Step 2: Run the contract tests and confirm they fail**

Run: `go test ./internal/cli -run 'TestAgents.*(Help|Remediation)' -count=1`

Expected: FAIL against the current local-only descriptions and add remediation.

- [ ] **Step 3: Correct user-facing command text**

Use these descriptions:

```go
Short: "Install a declared agent and record it"
Short: "Reconcile installed agents with their declared sources"
```

Change the doctor source-drift remediation from `agents add` to
`homonto agents update <name>`.

- [ ] **Step 4: Replace specification placeholders and stale catalog claims**

Give each `## Purpose` a one-paragraph capability definition. In
`tool-adapters/spec.md`, remove the statement that builtin lookup is not
implemented and state that builtin resources resolve from the versioned
materialized catalog. Do not edit archived change artifacts.

- [ ] **Step 5: Verify specs contain no placeholder purposes**

Run: `rg -n 'TBD - created by archiving' openspec/specs`

Expected: no output.

- [ ] **Step 6: Commit CLI and spec truth together**

```bash
git add internal/cli/agents.go internal/cli/agents_builtin_test.go openspec/specs
git commit -m "docs: align agent contracts and living specs"
```

### Task 4: Split the Docker Harness and Build Both Binaries

**Files:**
- Modify: `test/docker/Dockerfile`
- Rename: `test/docker/smoke.sh` to `test/docker/homonto-core.sh`
- Create: `test/docker/smoke.sh`
- Modify: `scripts/docker-test.sh`

**Interfaces:**
- Produces: `/usr/local/bin/homonto`, `/usr/local/bin/onto`, and a suite dispatcher accepting `all`, `homonto-core`, `homonto-expanded`, `homonto-agents`, or `onto-lifecycle`.

- [ ] **Step 1: Preserve the existing smoke as a named suite**

Run: `git mv test/docker/smoke.sh test/docker/homonto-core.sh`

Change its final line to `SMOKE PASS: homonto-core`.

- [ ] **Step 2: Build both binaries in the image**

Replace the build section in `test/docker/Dockerfile` with:

```dockerfile
RUN go build -o /usr/local/bin/homonto . \
 && go build -o /usr/local/bin/onto ./cmd/onto

ENTRYPOINT ["sh", "/src/test/docker/smoke.sh"]
```

- [ ] **Step 3: Create the suite dispatcher**

Create `test/docker/smoke.sh`:

```sh
#!/bin/sh
set -eu

suite="${1:-all}"
run_suite() { sh "/src/test/docker/$1.sh"; }

case "$suite" in
  all)
    run_suite homonto-core
    run_suite homonto-expanded
    run_suite homonto-agents
    run_suite onto-lifecycle
    ;;
  homonto-core|homonto-expanded|homonto-agents|onto-lifecycle)
    run_suite "$suite"
    ;;
  *)
    printf 'unknown docker suite: %s\n' "$suite" >&2
    exit 64
    ;;
esac
```

- [ ] **Step 4: Let the wrapper select a suite**

Change the final command in `scripts/docker-test.sh` to:

```sh
docker run --rm "$IMAGE" "${1:-all}"
```

- [ ] **Step 5: Verify the preserved core suite**

Run: `./scripts/docker-test.sh homonto-core`

Expected: `SMOKE PASS: homonto-core`.

- [ ] **Step 6: Commit the dual-binary harness**

```bash
git add scripts/docker-test.sh test/docker
git commit -m "test(e2e): build both binaries and split docker suites"
```

### Task 5: Add Expanded Homonto Projection E2E

**Files:**
- Create: `test/docker/homonto-expanded.sh`

**Interfaces:**
- Consumes: compiled `homonto`, embedded catalog, disposable HOME/workspace.
- Produces: binary-level evidence for framework, builtin resource, plugin, marketplace, and TUI projection.

- [ ] **Step 1: Create a disposable config with complete model routes**

Create `test/docker/homonto-expanded.sh` with this complete setup:

```sh
#!/bin/sh
set -eu

fail() { printf 'SMOKE FAIL: %s\n' "$1" >&2; exit 1; }
HOME="$(mktemp -d)"; export HOME
WORK="$(mktemp -d)"; cd "$WORK"

cat > homonto.toml <<'EOF'
[frameworks.comet]
source = "builtin:comet"
scope = "project"

[commands.example-command]
source = "builtin:example-command"
scope = "project"

[subagents.code-reviewer]
source = "builtin:code-reviewer"
scope = "project"

[marketplaces.claude.official]
source = "github"
repo = "anthropics/claude-plugins"

[plugins.claude.hud]
source = "claude-hud@official"
config = { compact = true }

[plugins.opencode.quota]
source = "@slkiser/opencode-quota"

[tui.opencode]
theme = "gruvbox"

[models.claude.architectural]
model = "opus"
variant = "max"
[models.claude.coding]
model = "sonnet"
effort = "normal"
[models.claude.trivial]
model = "haiku"
effort = "fast"
[models.opencode.architectural]
model = "anthropic/claude-opus-4-8"
effort = "high"
[models.opencode.coding]
model = "anthropic/claude-sonnet-4"
effort = "medium"
[models.opencode.trivial]
model = "openai/gpt-5-mini"
variant = "cheap"
EOF
```

- [ ] **Step 2: Apply through the compiled binary**

Run `homonto apply --yes`, then assert:

```sh
[ -d .homonto/catalog/skills/comet ] || fail "comet catalog not materialized"
[ -L .claude/commands/example-command.md ] || fail "claude command link missing"
[ -L .opencode/command/example-command.md ] || fail "opencode command link missing"
[ -L .claude/agents/code-reviewer.md ] || fail "claude subagent link missing"
[ -L .opencode/agent/code-reviewer.md ] || fail "opencode subagent link missing"
grep -q 'extraKnownMarketplaces' "$HOME/.claude/settings.json" || fail "marketplace missing"
grep -q 'pluginConfigs' "$HOME/.claude/settings.json" || fail "plugin config missing"
grep -q '@slkiser/opencode-quota' "$HOME/.config/opencode/opencode.jsonc" || fail "opencode plugin missing"
grep -q 'gruvbox' "$HOME/.config/opencode/tui.json" || fail "tui setting missing"
```

- [ ] **Step 3: Verify idempotency and health**

Append these exact assertions and the suite verdict:

```sh
out="$(homonto apply --yes 2>&1)"
printf '%s' "$out" | grep -q 'No changes' || fail "expanded apply is not idempotent"
homonto status | grep -q 'No drift' || fail "expanded status reports drift"
homonto doctor 2>&1 | grep -q 'command "example-command" linked (claude)' || fail "doctor missed claude command"
homonto doctor 2>&1 | grep -q 'subagent "code-reviewer" linked (opencode)' || fail "doctor missed opencode subagent"
printf 'SMOKE PASS: homonto-expanded\n'
```

- [ ] **Step 4: Run the suite**

Run: `./scripts/docker-test.sh homonto-expanded`

Expected: `SMOKE PASS: homonto-expanded`.

- [ ] **Step 5: Commit expanded projection evidence**

```bash
git add test/docker/homonto-expanded.sh
git commit -m "test(e2e): cover expanded homonto projection"
```

### Task 6: Add Homonto Agent Lifecycle E2E

**Files:**
- Create: `test/docker/homonto-agents.sh`

**Interfaces:**
- Consumes: compiled `homonto`, local agent source, builtin catalog agent.
- Produces: binary-level add/doctor/update/merge/conflict/prune evidence.

- [ ] **Step 1: Install local and builtin agents**

Create `test/docker/homonto-agents.sh` with this setup:

```sh
#!/bin/sh
set -eu
fail() { printf 'SMOKE FAIL: %s\n' "$1" >&2; exit 1; }
HOME="$(mktemp -d)"; export HOME
WORK="$(mktemp -d)"; cd "$WORK"
mkdir -p homonto/agents
printf 'line1\nline2\nline3\nline4\nline5\nline6\n' > homonto/agents/rev.md
printf 'one\ntwo\nthree\n' > homonto/agents/conflict.md

cat > homonto.toml <<'EOF'
[agents.rev]
source = "local:rev"
mode = "copy"
targets = ["claude", "opencode"]
[agents.conflict]
source = "local:conflict"
mode = "copy"
targets = ["claude"]
[agents.reviewer]
source = "builtin:code-reviewer"
targets = ["claude"]

[models.claude.architectural]
model = "opus"
variant = "max"
[models.claude.coding]
model = "sonnet"
effort = "normal"
[models.claude.trivial]
model = "haiku"
effort = "fast"
[models.opencode.architectural]
model = "anthropic/claude-opus-4-8"
effort = "high"
[models.opencode.coding]
model = "anthropic/claude-sonnet-4"
effort = "medium"
[models.opencode.trivial]
model = "openai/gpt-5-mini"
variant = "cheap"
EOF

homonto agents add rev
homonto agents add conflict
homonto agents add reviewer
[ -f .homonto/agents-lock.json ] || fail "agent lockfile missing"
[ -f "$HOME/.claude/agents/reviewer.md" ] || fail "builtin agent missing"
[ -n "$(ls -A .homonto/agents-blobs)" ] || fail "base blob store empty"
```

- [ ] **Step 2: Exercise a clean three-way merge**

Append:

```sh
REV="$HOME/.claude/agents/rev.md"
printf 'LOCAL1\nline2\nline3\nline4\nline5\nline6\n' > "$REV"
printf 'line1\nline2\nline3\nline4\nline5\nUPSTREAM6\n' > homonto/agents/rev.md
homonto agents update rev
grep -q 'LOCAL1' "$REV" || fail "clean merge lost local edit"
grep -q 'UPSTREAM6' "$REV" || fail "clean merge lost source edit"
[ -f "$REV.bak" ] || fail "clean merge backup missing"
```

- [ ] **Step 3: Exercise conflict reporting**

Append:

```sh
CONFLICT="$HOME/.claude/agents/conflict.md"
printf 'one\nLOCAL\nthree\n' > "$CONFLICT"
printf 'one\nUPSTREAM\nthree\n' > homonto/agents/conflict.md
if homonto agents update conflict; then fail "overlapping update succeeded"; fi
grep -q '^LOCAL$' "$CONFLICT" || fail "conflict changed live install"
grep -q '<<<<<<< local' "$CONFLICT.merged" || fail "conflict marker missing"
if homonto agents doctor >/tmp/agents-doctor.out 2>&1; then fail "doctor missed conflict"; fi
grep -q 'conflicted' /tmp/agents-doctor.out || fail "doctor conflict finding missing"
```

- [ ] **Step 4: Exercise dry-run and real prune**

Rewrite only the `rev` targets to `claude`, then append these assertions:

```sh
OPEN_REV="$HOME/.config/opencode/agent/rev.md"
sed -i '0,/targets = \["claude", "opencode"\]/s//targets = ["claude"]/' homonto.toml
homonto agents prune --dry-run
[ -e "$OPEN_REV" ] || fail "dry-run removed opencode target"
grep -q 'opencode' .homonto/agents-lock.json || fail "dry-run changed lockfile"
homonto agents prune
[ ! -e "$OPEN_REV" ] || fail "prune kept de-declared opencode target"
[ -e "$REV" ] || fail "prune removed retained claude target"
printf 'SMOKE PASS: homonto-agents\n'
```

- [ ] **Step 5: Run the suite**

Run: `./scripts/docker-test.sh homonto-agents`

Expected: `SMOKE PASS: homonto-agents`.

- [ ] **Step 6: Commit lifecycle E2E**

```bash
git add test/docker/homonto-agents.sh
git commit -m "test(e2e): cover homonto agent lifecycle"
```

### Task 7: Add Complete Onto Lifecycle E2E

**Files:**
- Create: `test/docker/onto-lifecycle.sh`

**Interfaces:**
- Consumes: compiled `homonto` and `onto`, project-scoped builtin Onto framework.
- Produces: binary-level framework gate, workflow transition, doctor, dependency, close, and archive evidence.

- [ ] **Step 1: Verify the framework gate refuses an uninitialized project**

Create `test/docker/onto-lifecycle.sh` with:

```sh
#!/bin/sh
set -eu
fail() { printf 'SMOKE FAIL: %s\n' "$1" >&2; exit 1; }
HOME="$(mktemp -d)"; export HOME
WORK="$(mktemp -d)"; cd "$WORK"
if onto init >/tmp/onto-gate.out 2>&1; then fail "onto init bypassed homonto gate"; fi
grep -q 'run `homonto init` first' /tmp/onto-gate.out || fail "missing init guidance"
git init -q
git config user.email smoke@example.invalid
git config user.name Smoke
```

- [ ] **Step 2: Install the Onto framework through Homonto**

Append this exact framework configuration and apply:

```sh
cat > homonto.toml <<'EOF'
[frameworks.onto]
source = "builtin:onto"
scope = "project"

[models.claude.architectural]
model = "opus"
variant = "max"
[models.claude.coding]
model = "sonnet"
effort = "normal"
[models.claude.trivial]
model = "haiku"
effort = "fast"
[models.opencode.architectural]
model = "anthropic/claude-opus-4-8"
effort = "high"
[models.opencode.coding]
model = "anthropic/claude-sonnet-4"
effort = "medium"
[models.opencode.trivial]
model = "openai/gpt-5-mini"
variant = "cheap"
EOF
homonto apply --yes
[ -d .homonto/catalog/skills/onto ] || fail "onto framework not materialized"
```

- [ ] **Step 3: Exercise init and change creation**

Append:

```sh
onto init
for d in changes specs adr guides; do [ -d "docs/$d" ] || fail "docs/$d missing"; done
onto new smoke-change
CHANGE="docs/changes/smoke-change"
for f in onto-state.yaml proposal.md tasks.md; do [ -f "$CHANGE/$f" ] || fail "$f missing"; done
onto status | grep -q 'smoke-change' || fail "status omitted change"
git add . && git commit -qm 'seed onto smoke'
```

- [ ] **Step 4: Exercise transition failure gates**

Append:

```sh
onto advance smoke-change
if onto advance smoke-change; then fail "design phase advanced without design.md"; fi
printf '# Design\n' > "$CHANGE/design.md"
onto advance smoke-change
printf '# Plan\n' > "$CHANGE/plan.md"
printf '%s\n' '- [ ] unfinished' > "$CHANGE/tasks.md"
if onto advance smoke-change; then fail "build advanced with unchecked task"; fi
printf '%s\n' '- [x] complete' > "$CHANGE/tasks.md"
onto advance smoke-change
printf '# Verification\nPass.\n' > "$CHANGE/verification.md"
git add . && git commit -qm 'prepare verify gate'
printf 'dirty\n' >> "$CHANGE/proposal.md"
if onto advance smoke-change; then fail "dirty worktree advanced to close"; fi
git restore "$CHANGE/proposal.md"
```

- [ ] **Step 5: Exercise the successful lifecycle**

Append:

```sh
onto advance smoke-change
git add . && git commit -qm 'advance smoke change to close'
onto close smoke-change
[ ! -d "$CHANGE" ] || fail "active change remains after close"
ARCHIVE="$(find docs/changes/archive -maxdepth 1 -type d -name '*-smoke-change' -print -quit)"
[ -n "$ARCHIVE" ] || fail "archive missing"
grep -q 'archived: true' "$ARCHIVE/onto-state.yaml" || fail "archive flag missing"
```

- [ ] **Step 6: Exercise doctor healthy and corrupt cases**

Append:

```sh
onto doctor
cp "$ARCHIVE/onto-state.yaml" /tmp/onto-state.yaml.clean
sed -i 's/archived: true/archived: false/' "$ARCHIVE/onto-state.yaml"
if onto doctor >/tmp/onto-doctor.out 2>&1; then fail "doctor accepted corrupt archive"; fi
grep -q 'archiv' /tmp/onto-doctor.out || fail "doctor did not name archive problem"
cp /tmp/onto-state.yaml.clean "$ARCHIVE/onto-state.yaml"
printf 'SMOKE PASS: onto-lifecycle\n'
```

- [ ] **Step 7: Run the suite**

Run: `./scripts/docker-test.sh onto-lifecycle`

Expected: `SMOKE PASS: onto-lifecycle`.

- [ ] **Step 8: Commit Onto E2E**

```bash
git add test/docker/onto-lifecycle.sh
git commit -m "test(e2e): cover complete onto lifecycle"
```

### Task 8: Add Release Artifact and Unified Gate Scripts

**Files:**
- Create: `scripts/test-release-artifacts.sh`
- Create: `scripts/verify-release.sh`
- Modify: `.github/workflows/ci.yml`
- Modify: `.github/workflows/release.yml`

**Interfaces:**
- Produces: `scripts/test-release-artifacts.sh <version>` verifies 12 archives and both stamped Linux binaries; `scripts/verify-release.sh` is the canonical local/release gate.

- [ ] **Step 1: Create archive assertions**

Create `scripts/test-release-artifacts.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail
VERSION="${1:?usage: test-release-artifacts.sh <version>}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
cd "$ROOT"
rm -rf dist
scripts/build-release.sh "$VERSION"
set -- dist/*.tar.gz dist/*.zip
[ "$#" -eq 12 ] || { printf 'expected 12 archives, found %s\n' "$#" >&2; exit 1; }
(cd dist && sha256sum -c SHA256SUMS)
tar -C "$TMP" -xzf "dist/homonto_${VERSION}_linux_amd64.tar.gz"
tar -C "$TMP" -xzf "dist/onto_${VERSION}_linux_amd64.tar.gz"
"$TMP/homonto_${VERSION}_linux_amd64/homonto" version 2>&1 | grep -q "$VERSION"
"$TMP/onto_${VERSION}_linux_amd64/onto" version 2>&1 | grep -q "$VERSION"
printf 'RELEASE ARTIFACTS PASS: %s\n' "$VERSION"
```

- [ ] **Step 2: Create the canonical full gate**

Create `scripts/verify-release.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

test -z "$(gofmt -l .)"
go mod tidy -diff
go vet ./...
go build ./...
go test ./... -count=1
go test -race ./... -count=1
./scripts/docker-test.sh all
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
./scripts/test-release-artifacts.sh "${1:-verification-dev}"
```

- [ ] **Step 3: Use named suites in CI**

Replace the single Docker job with a matrix over `homonto-core`,
`homonto-expanded`, `homonto-agents`, and `onto-lifecycle`. Add a packaging job
that runs `scripts/test-release-artifacts.sh ci-smoke`.

- [ ] **Step 4: Block release publication on the complete gate**

In `.github/workflows/release.yml`, run `scripts/verify-release.sh "${VERSION}"`
before publishing. Do not leave release publication dependent only on a
separate concurrently-running push workflow.

- [ ] **Step 5: Verify release scripts locally**

Run: `scripts/test-release-artifacts.sh verification-dev`

Expected: 12 verified archives, valid checksums, and both binaries report
`verification-dev`.

- [ ] **Step 6: Commit release gate unification**

```bash
git add scripts/test-release-artifacts.sh scripts/verify-release.sh .github/workflows/ci.yml .github/workflows/release.yml
git commit -m "ci: enforce dual-binary release verification"
```

### Task 9: Synchronize Release Documentation and Evidence

**Files:**
- Modify: `README.md`
- Modify: `docs/road-to-release.md`
- Modify: `docs/release-checklist.md`
- Modify: `docs/release-notes.md`
- Modify: `docs/roadmap.md`

**Interfaces:**
- Produces: one consistent release verdict and commands that match automation.

- [ ] **Step 1: Update the release checklist to call the canonical gate**

Replace the duplicated pre-tag command block with:

```sh
scripts/verify-release.sh v0.1.0-rc.1
```

Keep the post-tag external `go install` and downloaded-archive smoke because
those require a real public tag.

- [ ] **Step 2: Record completed evidence without changing the tag verdict**

Update road-to-release and the roadmap only after every new suite passes. Mark
dual-binary Docker and packaging evidence complete, but leave the
maintainer-owned tag and post-tag smoke unchecked.

- [ ] **Step 3: Check source/document agreement**

Run:

```bash
rg -n 'not implemented yet|not yet merged|168/168|168 tests|foundation.*only' README.md docs openspec/specs
```

Expected: matches only in clearly labeled historical/archive material, not
living user or release documents.

- [ ] **Step 4: Commit synchronized documentation**

```bash
git add README.md docs openspec/specs
git commit -m "docs: synchronize roadmap and release evidence"
```

### Task 10: Run the Full Release-Integrity Verification

**Files:**
- No production files expected.
- Update documentation only if verification reveals an inaccurate claim.

**Interfaces:**
- Produces: fresh evidence for the roadmap's Release Integrity exit gate.

- [ ] **Step 1: Run the canonical gate**

Run: `scripts/verify-release.sh verification-final`

Expected: formatting, tidy, vet, build, 443-or-more tests, race tests, all four
Docker suites, vulnerability scan, 12 archives, checksums, and both stamped
binaries pass.

- [ ] **Step 2: Verify the repository dogfood state**

Run: `go run . status`

Expected: `No drift.`

Run: `go run . doctor`

Expected: all declared skills, commands, and subagents linked; only documented
environmental warnings are allowed.

- [ ] **Step 3: Review the diff and roadmap gates**

Run: `git diff --check`

Expected: no whitespace errors.

Confirm each `Now: Release Integrity` exit gate in `docs/roadmap.md` has direct
test or command evidence. Do not mark the public tag complete.

- [ ] **Step 4: Commit verification-driven corrections, if any**

If verification required documentation-only corrections:

```bash
git add README.md docs openspec/specs
git commit -m "docs: record release integrity verification"
```

If no files changed, do not create an empty commit.

## Follow-up Plans Required

Do not implement the later roadmap phases from this master plan. Create one
focused design and implementation plan for each after the preceding exit gate:

1. Public stabilization and failure-path coverage.
2. Agent/subagent model reconciliation and migration.
3. Per-agent scope, compatibility metadata, conflict resolution, and blob GC.
4. Remote source trust model and threat controls.
5. Adapter contract and third-adapter pilot.
