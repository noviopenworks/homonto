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

- [x] 5.1 `testdata/` real `config.toml` with unmanaged keys + a user's own
  `[mcp_servers.other]`.
- [x] 5.2 Suite: apply projects the declared server; unmanaged key + user server
  preserved; second plan byte-identical (idempotent); de-declared server pruned
  while unmanaged survives; non-managed value never clobbered (conflict-safe).

## 6. Optional same-behavior migration (guarded)

- [x] 6.1 DEFERRED to follow-up: the exit gate (a third adapter without
  duplicated control flow) is met by Codex-on-contract. Migrating the
  heavily-tested Claude/OpenCode MCP slice onto structproj is a same-behavior
  refactor left as a tracked follow-up to avoid regression risk in this change.

## 7. Docs, ADR, gate

- [x] 7.1 ADR: adapter contract (format-agnostic projection core + Codec).
- [x] 7.2 Guide/README: Codex as a supported target; the adapter-contract shape.
- [x] 7.3 Roadmap item 11 → done (or partial with follow-up) with evidence.
- [x] 7.4 Full gate green: `go test -race ./...`, `./scripts/gate.sh`.

## Code review (review_mode: standard)

High-effort workflow review (base `9971404`) surfaced 7 findings. Fixed in
`1404ecd`: 3 correctness (codex accepted as target for kinds it can't project →
unloadable config; unescaped dotted MCP name → nested tables; delete recreates an
absent file) + 3 cleanups (Canonical dedup, EnsureRoot hoist, MustJSON export).
Accepted (deferred, not a defect): structproj being a "third copy" of the
plan/apply/observe semantics is the intended contract — migrating Claude/OpenCode
onto it is the tracked task-6 follow-up. No CRITICAL findings carried into verify.
