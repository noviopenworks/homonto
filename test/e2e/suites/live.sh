#!/usr/bin/env bash
# Suite: live — drive the REAL tool end-to-end against the user's own account
# (credentials mounted read-only at run time by the orchestrator) with a trivial
# prompt. Proves the installed tool actually runs and authenticates. Parametrized
# by $E2E_TOOL. opencode uses a free model so the check costs nothing.
set -uo pipefail
source /opt/e2e-suites/lib.sh
TOOL="${E2E_TOOL:?E2E_TOOL required}"
PROMPT='Reply with exactly the single word: PONG'

case "$TOOL" in
  claude)
    [ -f "$HOME/.claude/.credentials.json" ] || \
      fail "claude credentials not mounted (orchestrator must mount ~/.claude/.credentials.json read-only)"
    log "claude live prompt via mounted account"
    out="$(claude -p "$PROMPT" 2>&1)"; printf '%s\n' "$out"
    contains "$out" "PONG" "claude did not return PONG"
    pass "claude authenticated and returned PONG"
    ;;
  opencode)
    [ -f "$HOME/.local/share/opencode/auth.json" ] || \
      fail "opencode auth not mounted (orchestrator must mount ~/.local/share/opencode/auth.json read-only)"
    MODEL="${E2E_OPENCODE_MODEL:-opencode/north-mini-code-free}"
    log "opencode live prompt via mounted account (model: $MODEL)"
    out="$(opencode run -m "$MODEL" "$PROMPT" 2>&1)"; printf '%s\n' "$out"
    contains "$out" "PONG" "opencode did not return PONG"
    pass "opencode authenticated and returned PONG (free model, zero cost)"
    ;;
  *) fail "unknown E2E_TOOL: $TOOL" ;;
esac

printf '\nSUITE PASS: live/%s\n' "$TOOL"
