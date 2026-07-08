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

# --------------------------------------------- MCP + settings + secret refs
# Assertions here are against the projected FILES and the state file, not against
# homonto's stdout — the real proof that projection and secret resolution work.
log "mcp + settings + secret: apply projects into both tools"
MWORK="$(mktemp -d)"; cd "$MWORK"
SECRET_VALUE="smoke_do_not_leak_v4"
export SMOKE_SECRET="$SECRET_VALUE"
cat > homonto.toml <<'EOF'
[mcps.codegraph]
command = ["codegraph", "serve", "--mcp"]
targets = ["claude", "opencode"]
env = { API_KEY = "${SMOKE_SECRET}" }

[settings.claude]
model = "opus"

[settings.opencode]
theme = "opencode-dark"
EOF
homonto apply --yes

CJSON="$HOME/.claude.json"
CSET="$HOME/.claude/settings.json"
OJSONC="$HOME/.config/opencode/opencode.jsonc"
MSTATE="$MWORK/.homonto/state.json"

# Claude MCP server projected into .claude.json, with the secret RESOLVED.
grep -q '"codegraph"' "$CJSON"        || fail "claude mcp server not projected into .claude.json"
grep -q '"command": "codegraph"' "$CJSON" || fail "claude mcp command not projected"
grep -q "$SECRET_VALUE" "$CJSON"      || fail "secret env ref not resolved into claude .claude.json"
# Claude setting projected into settings.json.
grep -q 'opus' "$CSET"                || fail "claude setting not projected into settings.json"
# OpenCode MCP + setting projected into opencode.jsonc, secret resolved.
grep -q 'codegraph' "$OJSONC"         || fail "opencode mcp server not projected"
grep -q 'opencode-dark' "$OJSONC"     || fail "opencode setting not projected"
grep -q "$SECRET_VALUE" "$OJSONC"     || fail "secret env ref not resolved into opencode.jsonc"
# The state file must record the REFERENCE, never the resolved secret value.
grep -q 'SMOKE_SECRET' "$MSTATE"      || fail "state did not record the secret reference"
if grep -q "$SECRET_VALUE" "$MSTATE"; then fail "state LEAKED the resolved secret value"; fi

log "mcp + settings: second apply is idempotent"
out="$(homonto apply --yes 2>&1)"; printf '%s\n' "$out"
printf '%s' "$out" | grep -q "No changes" || fail "second mcp/settings apply was not idempotent"

# ------------------------------------------------------------- init command
log "init: scaffolds a fresh repo"
IWORK="$(mktemp -d)"
homonto init "$IWORK"
[ -f "$IWORK/homonto.toml" ]                 || fail "init did not create homonto.toml"
[ -f "$IWORK/.gitignore" ]                   || fail "init did not create .gitignore"
[ -f "$IWORK/content/skills/.gitkeep" ]      || fail "init did not create content/skills"

# ----------------------------------------------------------- import command
# ~/.claude.json now holds the codegraph server (from the MCP apply above), so
# import reads it back into a fresh homonto.toml.
log "import: bootstraps config, refuses overwrite without --force, then forces"
IMWORK="$(mktemp -d)"; cd "$IMWORK"
homonto import
grep -q 'codegraph' homonto.toml             || fail "import did not capture the claude mcp server"
printf '# manual edit\n' >> homonto.toml
homonto import 2>&1 | grep -q 'already exists' || fail "import without --force must refuse to overwrite"
grep -q '# manual edit' homonto.toml          || fail "import without --force overwrote the file"
homonto import --force
if grep -q '# manual edit' homonto.toml; then fail "import --force did not overwrite the file"; fi
grep -q 'codegraph' homonto.toml              || fail "import --force lost the imported server"

# ----------------------------------------------- conflict smoke (skill dirs)
# A real file or a foreign symlink where a skill link would go is user-owned:
# apply must ABORT and leave it byte-for-byte / target unchanged.
log "conflict: a real file at a skill dst aborts apply and is preserved"
XWORK="$(mktemp -d)"; cd "$XWORK"
mkdir -p content/skills/blocker
printf 'skill body\n' > content/skills/blocker/SKILL.md
printf '[skills]\nown = ["blocker"]\n' > homonto.toml
mkdir -p "$HOME/.claude/skills"
printf 'user data\n' > "$HOME/.claude/skills/blocker"
if homonto apply --yes >/dev/null 2>&1; then fail "apply must abort on a real file at the skill dst"; fi
grep -q 'user data' "$HOME/.claude/skills/blocker" || fail "apply clobbered the user's real file"
rm -f "$HOME/.claude/skills/blocker"

log "conflict: a foreign symlink at a skill dst aborts apply and is unchanged"
FOREIGN="$(mktemp -d)"
ln -s "$FOREIGN" "$HOME/.claude/skills/blocker"
if homonto apply --yes >/dev/null 2>&1; then fail "apply must abort on a foreign symlink at the skill dst"; fi
[ "$(readlink "$HOME/.claude/skills/blocker")" = "$FOREIGN" ] || fail "apply changed the foreign symlink"
rm -f "$HOME/.claude/skills/blocker"

printf '\nSMOKE PASS\n'
