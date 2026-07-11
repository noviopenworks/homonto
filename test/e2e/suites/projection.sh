#!/usr/bin/env bash
# Suite: projection — homonto projects config into a tool, then the REAL tool CLI
# is asked to read it back (`claude mcp list` / `opencode mcp list`). This proves
# homonto's output is consumed by the actual tool, not just written to disk.
# Parametrized by $E2E_TOOL (claude|opencode). No account/network needed.
set -uo pipefail
source /opt/e2e-suites/lib.sh
TOOL="${E2E_TOOL:?E2E_TOOL required}"

WORK="$(mktemp -d)"; cd "$WORK"
mkdir -p homonto/skills/e2e-demo
printf -- '---\nname: e2e-demo\ndescription: e2e projection skill\n---\nbody\n' \
  > homonto/skills/e2e-demo/SKILL.md

case "$TOOL" in
  claude)
    cat > homonto.toml <<'EOF'
[mcps.e2e-probe]
command = ["codegraph", "serve", "--mcp"]
targets = ["claude"]

[skills.e2e-demo]
source = "local:e2e-demo"
scope = "user"

[settings.claude]
model = "opus"
EOF
    log "homonto apply → claude"
    homonto apply --yes
    log "claude mcp list reads homonto's ~/.claude.json"
    out="$(claude mcp list 2>&1 || true)"; printf '%s\n' "$out"
    contains "$out" "e2e-probe" "claude mcp list did not show the homonto-projected server"
    pass "claude read the projected MCP server from ~/.claude.json"
    grep -q 'opus' "$HOME/.claude/settings.json" || fail "claude setting not projected into settings.json"
    pass "claude setting projected"
    link="$HOME/.claude/skills/e2e-demo"
    [ -L "$link" ] || fail "claude skill symlink not created"
    [ "$(readlink "$link")" = "$WORK/homonto/skills/e2e-demo" ] || fail "claude skill symlink target wrong"
    pass "claude skill symlinked to the owned source"
    ;;
  opencode)
    cat > homonto.toml <<'EOF'
[mcps.e2e-probe]
command = ["codegraph", "serve", "--mcp"]
targets = ["opencode"]

[skills.e2e-demo]
source = "local:e2e-demo"
scope = "user"

[settings.opencode]
theme = "opencode-dark"
EOF
    log "homonto apply → opencode"
    homonto apply --yes
    log "opencode mcp list reads homonto's opencode.jsonc"
    out="$(opencode mcp list 2>&1 || true)"; printf '%s\n' "$out"
    contains "$out" "e2e-probe" "opencode mcp list did not show the homonto-projected server"
    pass "opencode read the projected MCP server from opencode.jsonc"
    grep -q 'opencode-dark' "$HOME/.config/opencode/opencode.jsonc" || fail "opencode setting not projected"
    pass "opencode setting projected"
    link="$HOME/.config/opencode/skills/e2e-demo"
    [ -L "$link" ] || fail "opencode skill symlink not created"
    pass "opencode skill symlinked to the owned source"
    ;;
  *) fail "unknown E2E_TOOL: $TOOL" ;;
esac

log "second apply is idempotent"
out="$(homonto apply --yes 2>&1)"; printf '%s\n' "$out"
contains "$out" "No changes" "second apply was not idempotent"
pass "re-apply is a no-op"

log "homonto doctor confirms the links"
homonto doctor 2>&1 | tee /tmp/doctor.out
grep -q 'e2e-demo' /tmp/doctor.out || fail "doctor did not mention the owned skill"
pass "doctor healthy for the projected skill"

printf '\nSUITE PASS: projection/%s\n' "$TOOL"
