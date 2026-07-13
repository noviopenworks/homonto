#!/usr/bin/env bash
# The single, shared release/CI gate. Runs every check that must pass before a
# tag can publish, in one command, so local rehearsal, CI, and release
# publication all run the SAME gate — no path is weaker than another (closing the
# release-workflow-weaker-than-CI defect). Requires a Go toolchain and a Docker
# daemon (the dual-binary E2E suites build an image).
#
# Usage: scripts/gate.sh
set -euo pipefail
cd "$(dirname "$0")/.."

step() { printf '\n============================================================\n=== gate: %s\n============================================================\n' "$1"; }

step "gofmt -l"
unformatted="$(gofmt -l .)"
if [ -n "$unformatted" ]; then
  echo "these files are not gofmt-formatted:"; echo "$unformatted"; exit 1
fi

step "go mod tidy -diff"
go mod tidy -diff

step "go vet ./..."
go vet ./...

step "go build ./..."
go build ./...

step "go test ./..."
go test ./... -count=1

step "go test -race ./..."
go test -race ./... -count=1

step "version stamp smoke (homonto + onto)"
go build -ldflags "-X github.com/noviopenworks/homonto/internal/cli.Version=gate-smoke" -o /tmp/gate-homonto .
/tmp/gate-homonto version 2>&1 | grep -q "gate-smoke" || { echo "homonto version not stamped"; exit 1; }
go build -ldflags "-X github.com/noviopenworks/homonto/internal/ontocli.Version=gate-smoke" -o /tmp/gate-onto ./cmd/onto
/tmp/gate-onto version 2>&1 | grep -q "gate-smoke" || { echo "onto version not stamped"; exit 1; }

step "cli smoke (plan on a current-format config)"
printf '[mcps.demo]\ncommand = ["true"]\n' > /tmp/gate-homonto.toml
/tmp/gate-homonto --config /tmp/gate-homonto.toml plan >/dev/null

step "spec<->command correspondence"
./scripts/spec-command-check.sh

step "onto skills shell out (no direct state writes)"
./scripts/onto-skills-shell-out-check.sh

step "govulncheck ./..."
go run golang.org/x/vuln/cmd/govulncheck@latest ./...

step "dual-binary Docker E2E (five suites incl. release-packaging smoke)"
./scripts/docker-test.sh

printf '\n============================================================\nALL GATE CHECKS PASSED\n============================================================\n'
