# Tasks — adapter-registry

## 1. Registry package
- [ ] Add internal/adapter/registry: Deps, Factory, Registry (Register/Build in
      order, dup-panic), Builtins() registering claude/opencode/codex. Unit tests
      (Build yields the 3 in order; Register dup panics).

## 2. Engine wiring
- [ ] engine.Build constructs Deps and calls registry.Builtins().Build(deps);
      remove the hardcoded adapter literal. Engine + conformance suites green.

## 3. Verify
- [ ] `go test ./... -race`, vet, build, `openspec validate --all` green.
