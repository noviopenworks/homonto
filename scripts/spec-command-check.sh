#!/usr/bin/env bash
# Coarse spec<->code correspondence gate (ROADMAP N3 / F5).
#
# Fails if any canonical spec names a `homonto <command>` the CLI does not
# register. This is the exact failure that let agent-lifecycle/cli-commands
# mandate the removed `homonto agents` group while `openspec validate` still
# reported them valid: the validator checks form, not correspondence to reality.
#
# It is deliberately coarse — it guards the "spec names a removed command" class,
# not full semantic correspondence. Only backtick-delimited `homonto <cmd>`
# spans are inspected, so prose ("homonto reconciles ...") is never a command
# claim. Uppercase second words (`homonto SHALL ...`) are excluded by the token
# pattern.
set -euo pipefail
cd "$(dirname "$0")/.."

SPECS_DIR="${1:-openspec/specs}"
if [ ! -d "$SPECS_DIR" ]; then
  echo "spec-command-check: no $SPECS_DIR directory; nothing to check"
  exit 0
fi

# 1. Registered top-level commands, read from the actual binary so the check can
#    never drift from what the CLI really exposes. cobra always adds help and
#    completion.
bin="$(mktemp -u)/homonto"
mkdir -p "$(dirname "$bin")"
go build -o "$bin" . >/dev/null
registered="$("$bin" --help 2>/dev/null \
  | awk '/^Available Commands:/{f=1;next} /^Flags:/{f=0} f && NF {print $1}')"
registered="$registered help completion"
rm -rf "$(dirname "$bin")"

is_registered() {
  local cmd="$1" r
  for r in $registered; do [ "$cmd" = "$r" ] && return 0; done
  return 1
}

# 2. Every distinct `homonto <cmd>` naming a command, across the specs.
fails=0
while IFS=$'\t' read -r file cmd; do
  [ -z "$cmd" ] && continue
  if ! is_registered "$cmd"; then
    printf 'FAIL: %s names `homonto %s` but the CLI does not register `%s`\n' \
      "$file" "$cmd" "$cmd"
    fails=$((fails + 1))
  fi
done < <(
  grep -rHoE '`homonto [a-z][a-z-]+' "$SPECS_DIR" 2>/dev/null \
    | sed -E 's/^([^:]+):`homonto ([a-z-]+)$/\1\t\2/' \
    | sort -u
)

if [ "$fails" -gt 0 ]; then
  echo
  echo "spec-command-check FAILED: $fails stale command reference(s)."
  echo "Registered commands: $registered"
  exit 1
fi
echo "spec-command-check passed: every \`homonto <cmd>\` in $SPECS_DIR is registered."
