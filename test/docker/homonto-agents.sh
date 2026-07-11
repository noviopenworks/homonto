#!/bin/sh
# Suite: homonto-agents — the lifecycle-managed agent surface end to end: add,
# doctor, update (source refresh), clean three-way merge, conflict sidecar,
# dry-run prune, and prune. Asserts files, the lockfile, sidecars, and exit codes.
set -eu
SUITE=homonto-agents
. "$(dirname "$0")/lib.sh"

HOME="$(mktemp -d)"; export HOME
W="$(mktemp -d)"; cd "$W"
mkdir -p homonto/agents
printf 'alpha\nbeta\ngamma\n' > homonto/agents/rev.md
cat > homonto.toml <<'EOF'
[agents.rev]
source = "local:rev"
mode = "copy"
targets = ["claude"]
EOF
# Agents install at user scope (into $HOME); the lockfile lives beside the config.
DST="$HOME/.claude/agents/rev.md"
LOCK="$W/.homonto/agents-lock.json"

log "agents add installs and records the agent"
"$HOMONTO" agents add rev
is_file "$DST"
is_file "$LOCK"
in_file "$LOCK" '"rev"'
ok "installed + lockfile recorded"

log "agents doctor is healthy (read-only)"
"$HOMONTO" agents doctor
ok "doctor clean after add"

log "agents update re-materializes a changed source (no local edit)"
printf 'alpha\nbeta\ngamma v2\n' > homonto/agents/rev.md
"$HOMONTO" agents update rev
grep -q 'gamma v2' "$DST" || fail "update did not refresh the content"
absent "$DST.bak"
ok "source change re-materialized without a spurious backup"

log "clean three-way merge (disjoint local + source edits)"
printf 'ALPHA-LOCAL\nbeta\ngamma v2\n' > "$DST"          # local edits line 1
printf 'alpha\nbeta\nGAMMA-SOURCE\n' > homonto/agents/rev.md  # source edits line 3
"$HOMONTO" agents update rev
grep -q 'ALPHA-LOCAL'  "$DST" || fail "merge lost the local edit"
grep -q 'GAMMA-SOURCE' "$DST" || fail "merge lost the source edit"
absent "$DST.merged"
ok "disjoint edits auto-merged, no conflict sidecar"

log "conflicting edits produce a .merged sidecar and non-zero exit"
# base is now alpha/beta/GAMMA-SOURCE; both sides edit line 2 differently.
printf 'ALPHA-LOCAL\nbeta-LOCAL\nGAMMA-SOURCE\n'    > "$DST"
printf 'ALPHA-LOCAL\nbeta-UPSTREAM\nGAMMA-SOURCE\n' > homonto/agents/rev.md
before="$(cat "$DST")"
if "$HOMONTO" agents update rev >/tmp/agents-conflict.out 2>&1; then
  fail "a conflicting update must exit non-zero"
fi
is_file "$DST.merged"
[ "$(cat "$DST")" = "$before" ] || fail "the live file must be untouched on conflict"
in_file "$DST.merged" 'beta-LOCAL'
in_file "$DST.merged" 'beta-UPSTREAM'
ok "conflict wrote .merged with both sides; live file untouched"
rm -f "$DST.merged"

log "dry-run prune previews without changing anything"
printf '\n' > homonto.toml   # de-declare rev (now an orphan)
out="$("$HOMONTO" agents prune --dry-run 2>&1)"; printf '%s\n' "$out"
printf '%s' "$out" | grep -qi 'would remove\|dry run' || fail "dry-run must preview the removal"
is_file "$DST"
in_file "$LOCK" '"rev"'
ok "dry-run left the file and record intact"

log "prune removes the orphan and drops the record"
"$HOMONTO" agents prune
absent "$DST"
if grep -q '"rev"' "$LOCK" 2>/dev/null; then fail "prune did not drop the lockfile record"; fi
ok "prune removed the file and the record"

printf '\nSUITE PASS: %s\n' "$SUITE"
