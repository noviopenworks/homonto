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

# onto now ships code-reviewer + codebase-explorer as framework subagents, so
# declaring those explicitly would collide. Use comet-navigator (not shipped by
# onto) as the standalone explicit subagent that the prune test later removes.
[subagents.comet-navigator]
source = "builtin:comet-navigator"
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
is_file "$W/.homonto/catalog/commands/onto.md"
# onto ships two specialist subagents; comet-navigator is the explicit one.
is_file "$W/.homonto/catalog/subagents/code-reviewer.md"
is_file "$W/.homonto/catalog/subagents/codebase-explorer.md"
is_file "$W/.homonto/catalog/subagents/comet-navigator.md"
ok "framework skills, commands, and subagents materialized under .homonto/catalog"

# Assert each tool entry is a symlink AND that it actually resolves to real
# catalog content — a relative target computed against the wrong base dangles,
# and a dangling skill/command link is invisible to the tool (e.g. OpenCode's
# skill discovery skips it). is_dir/is_file follow the link, so they fail on a
# dangling target; link_to only string-matched and missed exactly that bug.
log "tool links point at (and resolve to) the materialized catalog"
is_link "$W/.claude/skills/onto";                 is_dir  "$W/.claude/skills/onto"
is_link "$W/.claude/agents/code-reviewer.md";     is_file "$W/.claude/agents/code-reviewer.md"
is_link "$W/.claude/agents/codebase-explorer.md"; is_file "$W/.claude/agents/codebase-explorer.md"
is_link "$W/.claude/agents/comet-navigator.md";   is_file "$W/.claude/agents/comet-navigator.md"
# The onto framework ships a command per phase/preset — the dispatcher plus every
# onto-* skill — so each phase is directly invocable. Assert the whole set links
# and resolves, not just the dispatcher.
for c in onto onto-open onto-design onto-build onto-verify onto-close onto-fix onto-tweak onto-no-slop; do
	is_link "$W/.claude/commands/$c.md"; is_file "$W/.claude/commands/$c.md"
done
ok "skill, full command set, and subagent links resolve to the catalog"

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

log "prune on removal: drop the explicit subagent, re-apply removes its link"
sed '/\[subagents.comet-navigator\]/,/targets = \["claude"\]/d' homonto.toml > homonto.toml.new
mv homonto.toml.new homonto.toml
"$HOMONTO" apply --yes >/dev/null 2>&1
absent "$W/.claude/agents/comet-navigator.md"
# The framework-provided subagents are NOT de-declared, so they must survive.
is_file "$W/.claude/agents/code-reviewer.md"
is_file "$W/.claude/agents/codebase-explorer.md"
ok "de-declared subagent link pruned; framework subagents retained"

printf '\nSUITE PASS: %s\n' "$SUITE"
