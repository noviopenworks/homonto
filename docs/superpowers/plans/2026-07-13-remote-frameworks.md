---
change: remote-frameworks
design-doc: docs/superpowers/specs/2026-07-13-remote-frameworks-design.md
base-ref: d6de69530dd7b1c0dfd08ed48e0cbf9db6eadd51
archived-with: 2026-07-13-remote-frameworks
---
# Plan
1. Config remote: accept + injected remoteFrameworkDirs + expansion. Engine
   resolveRemoteFrameworks (reuse remote.Resolver) injected at Build. E2E gate.
2. Verify: go test ./... -race, vet, build, openspec validate --all.
