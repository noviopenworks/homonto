#!/bin/sh
# Shared helpers for the test/docker dual-binary E2E suites. Each suite sources
# this, sets up its own disposable $HOME + workspace, and asserts against files,
# links, lockfiles, state, and exit codes (stdout matching only for output
# contracts). $HOMONTO / $ONTO select the binaries (defaults resolve on PATH), so
# a suite runs the same in-container and against locally built binaries.
: "${HOMONTO:=homonto}"
: "${ONTO:=onto}"

fail() { printf '\n  FAIL[%s]: %s\n' "${SUITE:-?}" "$1" >&2; exit 1; }
ok()   { printf '  ok: %s\n' "$1"; }
log()  { printf '\n=== [%s] %s ===\n' "${SUITE:-?}" "$1"; }

is_link() { [ -L "$1" ] || fail "expected symlink: $1"; }
is_dir()  { [ -d "$1" ] || fail "expected directory: $1"; }
is_file() { [ -f "$1" ] || fail "expected file: $1"; }
absent()  { [ ! -e "$1" ] || fail "expected absent: $1"; }
link_to() { [ "$(readlink "$1")" = "$2" ] || fail "link $1 -> $(readlink "$1"), want $2"; }
in_file() { grep -q -- "$2" "$1" || fail "$1 must contain '$2'"; }
