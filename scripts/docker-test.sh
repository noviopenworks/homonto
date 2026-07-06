#!/bin/sh
# Build the homonto e2e smoke image and run it. The container runs a real
# `apply` against a disposable $HOME (see test/docker/smoke.sh), so the host is
# never touched. Exits non-zero if the build or any smoke assertion fails.
set -eu

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
IMAGE="${HOMONTO_E2E_IMAGE:-homonto-e2e}"

cd "$ROOT"
docker build -f test/docker/Dockerfile -t "$IMAGE" .
docker run --rm "$IMAGE"
