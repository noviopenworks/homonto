## 1. TOML codec (internal/tomlutil)

- [x] 1.1 `Get(doc, dottedPath) (canonicalJSON, present)`, `Set(doc, path,
  jsonValue)`, `Delete(doc, path)`, `Canonical(jsonValue)`, `EnsureRoot(doc)`
  over TOML via go-toml/v2, preserving unmanaged tables/keys.
- [x] 1.2 Tests: set/get/delete round-trip; unmanaged keys preserved; canonical
  value form stable; empty/whitespace doc handled; nested `mcp_servers.<n>.env`.

## 2. Projection core / adapter contract (internal/adapter/structproj)

- [x] 2.1 `Codec` interface + `Project(desired, disk, st, ns, codec)` producing
  create/update/delete/noop/adopt changes (mirrors the Claude/OpenCode diff loop).
- [x] 2.2 `Apply(changes, disk, codec, res, st, ns)` writes only managed keys,
  records state, preserves unmanaged keys; `Observe(st, disk, ns, codec)`
  re-hashes recorded keys.
- [x] 2.3 Tests against a JSON codec (adapting jsonutil) proving the core
  reproduces create/update/noop/adopt/delete + secret redaction semantics.

## 3. Codex pilot adapter (internal/adapter/codex)

- [x] 3.1 Adapter: `Name()=="codex"`, `~/.codex/config.toml` path, `desired()`
  mapping declared MCPs → `mcp_servers.<name>` {command,args,env} (secret tokens
  preserved), Plan/Apply/ObserveHashes delegating to the core + TOML codec.
- [x] 3.2 Secret safety: plan redacts secret-bearing values; apply resolves only
  after confirm (reuse secret.Resolver). Test plan never prints a secret.

## 4. Config + engine wiring

- [x] 4.1 Add `codex` to the known tool set (target validation); keep default
  targets claude+opencode so existing configs are unaffected (codex is opt-in).
- [x] 4.2 Engine constructs the Codex adapter; plan/status/doctor include it.

## 5. Compatibility fixture suite

- [ ] 5.1 `testdata/` real `config.toml` with unmanaged keys + a user's own
  `[mcp_servers.other]`.
- [ ] 5.2 Suite: apply projects the declared server; unmanaged key + user server
  preserved; second plan byte-identical (idempotent); de-declared server pruned
  while unmanaged survives; non-managed value never clobbered (conflict-safe).

## 6. Optional same-behavior migration (guarded)

- [ ] 6.1 If it keeps every existing Claude/OpenCode test green, route their MCP
  structured-file projection through structproj to remove the duplication in
  place; otherwise leave them and record the migration as a follow-up.

## 7. Docs, ADR, gate

- [ ] 7.1 ADR: adapter contract (format-agnostic projection core + Codec).
- [ ] 7.2 Guide/README: Codex as a supported target; the adapter-contract shape.
- [ ] 7.3 Roadmap item 11 → done (or partial with follow-up) with evidence.
- [ ] 7.4 Full gate green: `go test -race ./...`, `./scripts/gate.sh`.
