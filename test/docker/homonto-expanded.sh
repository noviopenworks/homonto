#!/bin/sh
# Suite: homonto-expanded — builtin framework materialization, skill/command/
# subagent links into the tool dirs, plugin + marketplace projection into Claude
# settings, and OpenCode TUI projection, all against a disposable $HOME.
set -eu
SUITE=homonto-expanded
. "$(dirname "$0")/lib.sh"

HOME="$(mktemp -d)"; export HOME
W="$(mktemp -d)"; cd "$W"

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

[subagents.code-reviewer]
source = "builtin:code-reviewer"
scope = "project"
targets = ["claude"]

[marketplaces.claude.official]
source = "github"
repo = "anthropics/claude-plugins"

[plugins.claude.claude-hud]
source = "claude-hud@official"

[tui.opencode]
theme = "gruvbox"
EOF

log "apply projects the expanded surface"
"$HOMONTO" apply --yes

log "builtin catalog materialization"
is_dir  "$W/.homonto/catalog/skills/onto"
is_file "$W/.homonto/catalog/commands/example-command.md"
is_file "$W/.homonto/catalog/subagents/code-reviewer.md"
ok "framework skills, command, and subagent materialized under .homonto/catalog"

log "tool links point at the materialized catalog"
is_link "$W/.claude/skills/onto"
link_to "$W/.claude/skills/onto" ".homonto/catalog/skills/onto"
is_link "$W/.claude/commands/example-command.md"
link_to "$W/.claude/commands/example-command.md" ".homonto/catalog/commands/example-command.md"
is_link "$W/.claude/agents/code-reviewer.md"
link_to "$W/.claude/agents/code-reviewer.md" ".homonto/catalog/subagents/code-reviewer.md"
ok "skill, command, and subagent links resolve to the catalog"

log "plugin + marketplace projected into claude settings.json"
CSET="$HOME/.claude/settings.json"
is_file "$CSET"
in_file "$CSET" '"official"'
in_file "$CSET" 'claude-hud@official'
ok "extraKnownMarketplaces + enabledPlugins present"

log "opencode TUI projected into tui.json"
in_file "$HOME/.config/opencode/tui.json" 'gruvbox'
ok "tui.json theme projected"

log "re-apply is idempotent"
out="$("$HOMONTO" apply --yes 2>&1)"; printf '%s\n' "$out"
printf '%s' "$out" | grep -q "No changes" || fail "second apply was not idempotent"
ok "idempotent re-apply"

log "prune on removal: drop the subagent, re-apply removes its link"
sed '/\[subagents.code-reviewer\]/,/targets = \["claude"\]/d' homonto.toml > homonto.toml.new
mv homonto.toml.new homonto.toml
"$HOMONTO" apply --yes >/dev/null 2>&1
absent "$W/.claude/agents/code-reviewer.md"
ok "de-declared subagent link pruned on apply"

printf '\nSUITE PASS: %s\n' "$SUITE"
