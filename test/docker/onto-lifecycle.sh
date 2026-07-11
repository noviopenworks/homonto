#!/bin/sh
# Suite: onto-lifecycle — the onto binary end to end against a real materialized
# framework install: the framework-install gate, init, new, gated phase advances
# (including a failure gate), doctor, dependency-aware close, and archive.
set -eu
SUITE=onto-lifecycle
. "$(dirname "$0")/lib.sh"

HOME="$(mktemp -d)"; export HOME
W="$(mktemp -d)"; cd "$W"
git init -q
git config user.email e2e@example.com
git config user.name e2e

log "framework-install gate: onto init refuses before homonto apply"
cat > homonto.toml <<'EOF'
[frameworks.onto]
source = "builtin:onto"
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
if "$ONTO" init >/dev/null 2>&1; then fail "onto init must refuse before the framework is applied"; fi
absent "$W/docs"
ok "onto init refused and created no docs/ tree"

log "homonto apply installs the onto framework"
"$HOMONTO" apply --yes >/dev/null
is_dir "$W/.homonto/catalog/skills/onto"
ok "framework materialized"

log "onto init scaffolds the workspace"
"$ONTO" init >/dev/null
for d in changes specs adr guides; do is_dir "$W/docs/$d"; done
ok "docs/{changes,specs,adr,guides} created"

log "onto new creates an open-phase change"
"$ONTO" new feat-a >/dev/null
CH="$W/docs/changes/feat-a"
is_file "$CH/onto-state.yaml"; is_file "$CH/proposal.md"; is_file "$CH/tasks.md"
in_file "$CH/onto-state.yaml" 'phase: open'
ok "open-phase skeleton created"

log "advance open -> design, then a failure gate stops design -> build"
"$ONTO" advance feat-a >/dev/null
in_file "$CH/onto-state.yaml" 'phase: design'
if "$ONTO" advance feat-a >/dev/null 2>&1; then fail "advance must refuse to leave design without design.md"; fi
in_file "$CH/onto-state.yaml" 'phase: design'
ok "advance gated on the missing design.md deliverable"

log "produce deliverables and advance design -> build -> verify -> close"
printf '# Design\n' > "$CH/design.md"
"$ONTO" advance feat-a >/dev/null; in_file "$CH/onto-state.yaml" 'phase: build'
printf '# Plan\n' > "$CH/plan.md"
printf -- '- [x] done\n' > "$CH/tasks.md"
"$ONTO" advance feat-a >/dev/null; in_file "$CH/onto-state.yaml" 'phase: verify'
printf '# Verification\n' > "$CH/verification.md"
git add -A && git commit -q -m "feat-a artifacts"
"$ONTO" advance feat-a >/dev/null; in_file "$CH/onto-state.yaml" 'phase: close'
git add -A && git commit -q -m "feat-a enters close"
ok "feat-a advanced through every gate to close"

log "dependency-aware close: feat-b depends on the still-active feat-a"
"$ONTO" new feat-b >/dev/null
sed 's/^phase: open/phase: close/' "$W/docs/changes/feat-b/onto-state.yaml" > /tmp/fb.yaml
mv /tmp/fb.yaml "$W/docs/changes/feat-b/onto-state.yaml"
printf 'deps:\n- feat-a\n' >> "$W/docs/changes/feat-b/onto-state.yaml"
git add -A && git commit -q -m "feat-b at close depending on feat-a"
if "$ONTO" close feat-b >/dev/null 2>&1; then fail "close must refuse while dependency feat-a is unresolved"; fi
is_dir "$W/docs/changes/feat-b"
ok "close refused with an unresolved dependency"

log "close feat-a (archives it), then feat-b's dependency is satisfied"
"$ONTO" close feat-a >/dev/null
ARCH="$(find "$W/docs/changes/archive" -maxdepth 1 -name '*-feat-a' -type d | head -1)"
[ -n "$ARCH" ] || fail "feat-a was not archived"
in_file "$ARCH/onto-state.yaml" 'archived: true'
absent "$CH"
git add -A && git commit -q -m "archive feat-a"
"$ONTO" close feat-b >/dev/null
[ -n "$(find "$W/docs/changes/archive" -maxdepth 1 -name '*-feat-b' -type d)" ] || fail "feat-b did not archive after its dependency resolved"
ok "feat-a archived; feat-b closed once its dependency resolved"

log "onto doctor is healthy after clean closes"
git add -A && git commit -q -m "archive feat-b" || true
"$ONTO" doctor >/dev/null 2>&1 || fail "onto doctor reported problems"
ok "onto doctor healthy"

printf '\nSUITE PASS: %s\n' "$SUITE"
