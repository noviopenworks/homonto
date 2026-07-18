#!/bin/sh
# Suite: to-lifecycle — the to binary end to end against a real materialized
# framework install: the framework-install gate, init, new, the single phase
# advance, the required-but-self-asserted --verified flag, abandon, archive,
# the config-free read-only commands, and the onto-xor-to exclusivity error.
set -eu
SUITE=to-lifecycle
. "$(dirname "$0")/lib.sh"

HOME="$(mktemp -d)"; export HOME
W="$(mktemp -d)"; cd "$W"

log "framework-install gate: to init refuses before homonto apply"
cat > homonto.toml <<'EOF'
[frameworks.to]
source = "builtin:to"
scope = "project"
targets = ["claude"]

[models.claude.architectural]
model = "opus"
variant = "max"
[models.claude.coding]
model = "sonnet"
variant = "max"
[models.claude.trivial]
model = "haiku"
variant = "max"
EOF
if "$TO" init >/dev/null 2>&1; then fail "to init must refuse before the framework is applied"; fi
absent "$W/docs"
ok "to init refused and created no docs/ tree"

log "read-only commands are config-independent even before apply"
"$TO" status >/dev/null || fail "to status must work without an applied framework"
ok "to status answered without the framework"

log "onto and to are mutually exclusive in one config"
cp homonto.toml /tmp/to-only.toml
printf '\n[frameworks.onto]\nsource = "builtin:onto"\nscope = "project"\n' >> homonto.toml
if "$HOMONTO" plan >/dev/null 2>&1; then fail "homonto must refuse a config declaring both onto and to"; fi
cp /tmp/to-only.toml homonto.toml
ok "homonto refused the onto+to config"

log "homonto apply installs the to framework"
"$HOMONTO" apply --yes >/dev/null
is_dir "$W/.homonto/catalog/skills/to"
is_file "$W/.homonto/catalog/subagents/to-skeptic.md"
ok "framework materialized (skills + subagents)"

log "to init scaffolds docs/tasks + archive"
"$TO" init >/dev/null
is_dir "$W/docs/tasks"; is_dir "$W/docs/tasks/archive"
ok "docs/tasks and docs/tasks/archive created"

log "to new creates a plan-phase change with an empty plan.md"
"$TO" new feat-a >/dev/null
CH="$W/docs/tasks/feat-a"
is_file "$CH/to-state.yaml"; is_file "$CH/plan.md"
in_file "$CH/to-state.yaml" 'phase: plan'
ok "plan-phase skeleton created"

log "done refuses from plan and without --verified; phase advances plan -> do"
if "$TO" done feat-a --verified >/dev/null 2>&1; then fail "done must refuse from plan"; fi
"$TO" phase feat-a >/dev/null
in_file "$CH/to-state.yaml" 'phase: do'
if "$TO" done feat-a >/dev/null 2>&1; then fail "done must refuse without --verified"; fi
if "$TO" phase feat-a >/dev/null 2>&1; then fail "phase must refuse from do (done is the only exit)"; fi
ok "the one legal advance ran; done gated on --verified"

log "handoff prints the recovery pack"
printf '# plan\n- [ ] step\n' > "$CH/plan.md"
# CLI output goes to stderr (documented caveat) — fold it for the grep.
"$TO" handoff feat-a 2>&1 | grep -q 'phase: do' || fail "handoff must report the phase"
ok "handoff reported phase + plan"

log "done --verified archives; terminal is terminal"
"$TO" done feat-a --verified >/dev/null
ARCH="$W/docs/tasks/archive/feat-a"
is_dir "$ARCH"; absent "$CH"
in_file "$ARCH/to-state.yaml" 'phase: done'
in_file "$ARCH/to-state.yaml" 'verified: true'
if "$TO" phase feat-a >/dev/null 2>&1; then fail "phase must refuse on an archived change"; fi
if "$TO" new feat-a >/dev/null 2>&1; then fail "new must refuse to reuse an archived name"; fi
ok "feat-a done, archived, terminal"

log "abandon is the terminal exit without done"
"$TO" new feat-b >/dev/null
"$TO" abandon feat-b >/dev/null
in_file "$W/docs/tasks/archive/feat-b/to-state.yaml" 'phase: abandoned'
ok "feat-b abandoned and archived"

log "status --json lists nothing active after both archives"
"$TO" status --json 2>&1 | grep -q '^\[\]' || fail "status --json should be an empty array"
ok "active listing empty"

printf '\nSUITE PASS: %s\n' "$SUITE"
