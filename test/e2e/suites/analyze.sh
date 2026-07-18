#!/usr/bin/env bash
# Suite: analyze — not a pass/fail test but a structured dump of the container's
# internal state after a representative homonto apply (both tools + the onto
# framework). The orchestrator captures this to a file so you can inspect exactly
# what homonto wrote inside the image. Contains no secrets (state stores hashes).
set -uo pipefail
source /opt/e2e-suites/lib.sh

WORK="$(mktemp -d)"; cd "$WORK"
git init -q; git config user.email e2e@example.com; git config user.name e2e
mkdir -p homonto/skills/e2e-demo
printf -- '---\nname: e2e-demo\ndescription: d\n---\nbody\n' > homonto/skills/e2e-demo/SKILL.md
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

[models.claude.review]
model = "opus"
variant = "max"

[models.claude.trivial]
model = "haiku"
variant = "max"

[mcps.e2e-probe]
command = ["codegraph", "serve", "--mcp"]

[skills.e2e-demo]
source = "local:e2e-demo"
scope = "user"

[settings.claude]
model = "opus"

[settings.opencode]
theme = "opencode-dark"
EOF
homonto apply --yes >/dev/null 2>&1 || homonto apply --yes

echo "##################### CONTAINER INTERNALS ANALYSIS #####################"
echo
echo "## 1. installed binaries"
# homonto/onto route cobra output to stderr, so capture 2>&1.
printf '   homonto   %s\n' "$(homonto version 2>&1)"
printf '   onto      %s\n' "$(onto version 2>&1)"
printf '   claude    %s\n' "$(claude --version 2>/dev/null)"
printf '   opencode  %s\n' "$(opencode --version 2>/dev/null)"

echo
echo "## 2. homonto-managed tool config files (what apply wrote)"
for f in "$HOME/.claude.json" "$HOME/.claude/settings.json" "$HOME/.config/opencode/opencode.jsonc"; do
  if [ -f "$f" ]; then echo "   --- $f ---"; sed -e 's/^/       /' "$f" | head -30; fi
done

echo
echo "## 3. projected skill symlinks (owned content, linked not copied)"
find "$HOME/.claude/skills" "$HOME/.config/opencode/skills" -maxdepth 1 -type l \
  -printf '   %p -> %l\n' 2>/dev/null || true

echo
echo "## 4. materialized builtin catalog (.homonto/catalog)"
find "$WORK/.homonto/catalog" -maxdepth 2 -mindepth 1 -type d -printf '   %P\n' 2>/dev/null | sort | head -40

echo
echo "## 5. homonto state (unresolved refs + hashes, never plaintext secrets)"
sed -e 's/^/   /' "$WORK/.homonto/state.json" 2>/dev/null | head -40

echo
echo "## 6. the real tools see homonto's projection"
echo "   -- claude mcp list --"
claude mcp list 2>&1 | sed -e 's/^/   /' | head -6
echo "   -- opencode mcp list --"
opencode mcp list 2>&1 | sed -e 's/^/   /' | head -10

echo
echo "##################### END ANALYSIS #####################"
