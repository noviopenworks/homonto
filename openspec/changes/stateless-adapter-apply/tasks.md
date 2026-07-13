# Tasks — stateless-adapter-apply

## 1. Interface + adapters
- [x] adapter.Adapter.Apply gains a leading cfg *config.Config param. Each of
      claude/opencode extracts an expand(cfg) helper called at the top of both
      Plan and Apply; codex accepts and ignores cfg. engine.Apply passes e.Cfg.

## 2. Test call sites
- [x] Update the ~61 direct adapter Apply call sites to pass the config already
      in scope (from the preceding Plan). No assertion changes.

## 3. Verify
- [x] `go test ./... -race`, vet, build, `openspec validate --all` green;
      conformance + all adapter/engine tests pass unchanged.
