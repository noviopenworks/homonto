#!/bin/sh
# End-to-end smoke test for the compiled homonto binary. Runs a real
# `apply` against a throwaway $HOME and workspace, so it exercises the actual
# os.UserHomeDir() code path (not the test-only HOME override) without touching
# any real system. Meant to run inside the Docker image (test/docker/Dockerfile).
set -eu

fail() { printf '\nSMOKE FAIL: %s\n' "$1" >&2; exit 1; }
log()  { printf '\n=== %s ===\n' "$1"; }

# Disposable HOME and workspace — everything homonto writes lands here.
HOME="$(mktemp -d)"
export HOME
WORK="$(mktemp -d)"
cd "$WORK"

# A minimal owned skill for homonto to link.
mkdir -p content/skills/demo
cat > content/skills/demo/SKILL.md <<'EOF'
---
name: demo
description: smoke-test skill
---
demo skill body
EOF

CLAUDE_USER="$HOME/.claude/skills/demo"
OPEN_USER="$HOME/.config/opencode/skills/demo"
CLAUDE_PROJ="$WORK/.claude/skills/demo"
OPEN_PROJ="$WORK/.opencode/skills/demo"
SRC="$WORK/content/skills/demo"

# ---------------------------------------------------------------- user scope
log "user scope: apply"
printf '[skills]\nown = ["demo"]\n' > homonto.toml
homonto apply --yes

[ -L "$CLAUDE_USER" ] || fail "claude user link not created"
[ -L "$OPEN_USER" ]   || fail "opencode user link not created"
[ "$(readlink "$CLAUDE_USER")" = "$SRC" ] || fail "claude link points at $(readlink "$CLAUDE_USER"), want $SRC"
[ -e "$CLAUDE_PROJ" ] && fail "user scope must not create a project link"

log "user scope: second apply is idempotent"
out="$(homonto apply --yes 2>&1)"
printf '%s\n' "$out"
printf '%s' "$out" | grep -q "No changes" || fail "second user apply was not idempotent"

log "user scope: status + doctor"
homonto status
homonto doctor 2>&1 | tee /tmp/doctor.out
grep -q 'ok: skill "demo" linked (claude)'   /tmp/doctor.out || fail "doctor did not confirm claude link"
grep -q 'ok: skill "demo" linked (opencode)' /tmp/doctor.out || fail "doctor did not confirm opencode link"

# ------------------------------------------------------------- project scope
log "project scope: apply relocates links into the repo"
printf '[skills]\nscope = "project"\nown = ["demo"]\n' > homonto.toml
homonto apply --yes

[ -L "$CLAUDE_PROJ" ] || fail "claude project link not created"
[ -L "$OPEN_PROJ" ]   || fail "opencode project link not created"
[ "$(readlink "$CLAUDE_PROJ")" = "$SRC" ] || fail "project claude link points at wrong target"
# The old user-scope links must have been pruned — no orphan.
[ -e "$CLAUDE_USER" ] && fail "user claude link not pruned after switch to project"
[ -e "$OPEN_USER" ]   && fail "user opencode link not pruned after switch to project"

log "project scope: second apply is idempotent"
out="$(homonto apply --yes 2>&1)"
printf '%s\n' "$out"
printf '%s' "$out" | grep -q "No changes" || fail "second project apply was not idempotent"

log "project scope: doctor checks the project location"
homonto doctor 2>&1 | tee /tmp/doctor2.out
grep -q 'ok: skill "demo" linked (claude)'   /tmp/doctor2.out || fail "doctor did not confirm claude project link"
grep -q 'ok: skill "demo" linked (opencode)' /tmp/doctor2.out || fail "doctor did not confirm opencode project link"

printf '\nSMOKE PASS\n'
