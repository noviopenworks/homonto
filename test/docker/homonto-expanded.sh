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
variant = "1m"
effort = "high"
[models.claude.coding]
model = "sonnet"
effort = "medium"
[models.claude.review]
model = "opus"
variant = "1m"
effort = "high"
[models.claude.trivial]
model = "haiku"
effort = "low"

# Retune one agent: wins over its tier field by field, and needs no source
# because the onto framework already declares this agent.
[subagents.onto-skeptic.claude]
effort = "max"

# onto ships all builtin subagents as framework subagents, so declaring one
# explicitly would collide. Use a local: agent file (homonto/subagents/) as the
# standalone explicit subagent that the prune test later removes.
[subagents.nav-agent]
source = "local:nav-agent"
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

mkdir -p homonto/subagents
cat > homonto/subagents/nav-agent.md <<'AGENT'
---
description: local standalone test agent
---
Route the user to the right change.
AGENT

log "apply projects the expanded surface"
"$HOMONTO" apply --yes

log "builtin catalog materialization"
is_dir  "$W/.homonto/catalog/skills/onto"
is_file "$W/.homonto/catalog/commands/onto.md"
# onto ships four specialist subagents; nav-agent is the explicit local one
# (local sources link straight from homonto/subagents/, no materialization).
is_file "$W/.homonto/catalog/subagents/onto-reviewer.md"
is_file "$W/.homonto/catalog/subagents/onto-explorer.md"
is_file "$W/.homonto/catalog/subagents/onto-implementer.md"
is_file "$W/.homonto/catalog/subagents/onto-skeptic.md"
# Homonto-block subagents materialize per-tool variants; the Claude variant of a
# read-only spawn:[] agent denies exactly the removed capabilities (everything
# else keeps Claude's defaults), and stamps the role's model (this config maps
# the claude review tier -> opus).
in_file "$W/.homonto/catalog/subagents/onto-reviewer.claude.md" 'disallowedTools: Edit, Write, NotebookEdit, Agent, Task'
if grep -qE '^tools:|^mode:' "$W/.homonto/catalog/subagents/onto-reviewer.claude.md"; then fail "claude render must not carry a tools: allowlist or mode: field"; fi
# Claude has no variant field: a variant brackets the ALIAS into the model, and
# effort is its own field. Both come from the review tier.
in_file "$W/.homonto/catalog/subagents/onto-reviewer.claude.md" 'model: opus\[1m\]'
in_file "$W/.homonto/catalog/subagents/onto-reviewer.claude.md" 'effort: high'
# A per-subagent override beats the tier field by field: the skeptic shares the
# review tier but thinks at max, and still inherits that tier's model.
in_file "$W/.homonto/catalog/subagents/onto-skeptic.claude.md" 'effort: max'
in_file "$W/.homonto/catalog/subagents/onto-skeptic.claude.md" 'model: opus\[1m\]'
# The implementer edits (coding model) but still spawns nothing: the only
# denial is spawning — Edit/Write stay available (absent from the denylist).
in_file "$W/.homonto/catalog/subagents/onto-implementer.claude.md" 'model: sonnet'
in_file "$W/.homonto/catalog/subagents/onto-implementer.claude.md" 'effort: medium'
in_file "$W/.homonto/catalog/subagents/onto-implementer.claude.md" 'disallowedTools: Agent, Task'
if grep -qE 'disallowedTools:.*Edit' "$W/.homonto/catalog/subagents/onto-implementer.claude.md"; then fail "edit-capable implementer must not deny Edit"; fi
# The onto primary agent is OpenCode-only: agentfm skips its Claude render, so
# the .claude.md variant is absent while the .opencode.md variant exists.
is_file "$W/.homonto/catalog/subagents/onto.md"
is_file "$W/.homonto/catalog/subagents/onto.opencode.md"
absent  "$W/.homonto/catalog/subagents/onto.claude.md"
ok "framework skills, commands, and subagents materialized (per-tool render invariants hold)"

# Assert each tool entry is a symlink AND that it actually resolves to real
# catalog content — a relative target computed against the wrong base dangles,
# and a dangling skill/command link is invisible to the tool (e.g. OpenCode's
# skill discovery skips it). is_dir/is_file follow the link, so they fail on a
# dangling target; link_to only string-matched and missed exactly that bug.
log "tool links point at (and resolve to) the materialized catalog"
is_link "$W/.claude/skills/onto";                 is_dir  "$W/.claude/skills/onto"
is_link "$W/.claude/agents/onto-reviewer.md";     is_file "$W/.claude/agents/onto-reviewer.md"
is_link "$W/.claude/agents/onto-explorer.md"; is_file "$W/.claude/agents/onto-explorer.md"
is_link "$W/.claude/agents/onto-implementer.md";  is_file "$W/.claude/agents/onto-implementer.md"
is_link "$W/.claude/agents/nav-agent.md";         is_file "$W/.claude/agents/nav-agent.md"
# The onto primary agent has no Claude variant, so it is NOT projected for Claude
# (its entry point is the /onto command → onto skill).
absent "$W/.claude/agents/onto.md"
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
sed '/\[subagents.nav-agent\]/,/targets = \["claude"\]/d' homonto.toml > homonto.toml.new
mv homonto.toml.new homonto.toml
"$HOMONTO" apply --yes >/dev/null 2>&1
absent "$W/.claude/agents/nav-agent.md"
# The framework-provided subagents are NOT de-declared, so they must survive.
is_file "$W/.claude/agents/onto-reviewer.md"
is_file "$W/.claude/agents/onto-explorer.md"
is_file "$W/.claude/agents/onto-implementer.md"
ok "de-declared subagent link pruned; framework subagents retained"

printf '\nSUITE PASS: %s\n' "$SUITE"
