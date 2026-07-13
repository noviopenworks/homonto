# Comet Design Handoff

- Change: config-load-phases
- Phase: design
- Mode: compact
- Context hash: f8e2b72aa3bf2f0c81b61b6701263dae00d11e4dd3621553a574912f18b68f02

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/config-load-phases/proposal.md

- Source: openspec/changes/config-load-phases/proposal.md
- Lines: 1-43
- SHA256: 5bf35732f6c140069d9cebfcfa5ea033d144e60b97663ae01d7c616cd0f0e42a

```md
# Split config.Load into explicit decode → migrate → normalize → validate phases

## Why

Roadmap X3 (F43). `config.Load` is a ~200-line monolith that interleaves reading,
TOML decoding, the schema-version guard, the `[agents]`→`[subagents]` fold, scope
normalization, and a large inline block of validation. The X3 exit gate calls for
config loading to split "into explicit phases (decode → migrate → normalize →
validate → expand)… ending the monolith."

## What Changes

Extract `Load`'s existing steps — **in the same order, with no behavior change** —
into named phase functions:

- `decode([]byte) (*Config, error)` — TOML unmarshal + the schema-version
  forward-safety guard.
- `migrate(*Config)` — the `[agents]`→copy-mode `[subagents]` fold (Option C).
- `normalize(*Config)` — subagent scope defaulting.
- `validate(*Config) error` — the resource/framework/subagent/model/MCP/plugin/
  marketplace validation block (unchanged, same order).

`Load` becomes: read file → `decode` → `migrate` → `normalize` → `validate` →
return. Each phase is now individually legible and testable.

Expansion (`Expanded*EntriesForTool`) is left as-is — unifying it into a generic
per-kind pipeline is a larger follow-on; this slice ends the `Load` monolith.

## Impact

- **Specs:** `config-model` gains a requirement that config loading runs as
  explicit ordered phases (decode → migrate → normalize → validate).
- **Behavior:** none — a pure in-order extract-method refactor; every load,
  validation error, and fold behaves exactly as before, pinned by the config
  suite.
- **Risk:** low — mechanical extraction with no reordering; the comprehensive
  config load/validation tests are the safety net.

## Non-goals

- The generic per-kind expansion pipeline (the "expand" phase) — a larger
  follow-on.
- Any validation-rule or behavior change.

```

## openspec/changes/config-load-phases/design.md

- Source: openspec/changes/config-load-phases/design.md
- Lines: 1-47
- SHA256: b9aea69e2838a4d27bc98ea4a0ae974150df2c6bb1704abd2e185ffeea14c9bb

```md
# Design — config.Load phase split

## Approach

Pure in-order extract-method. Move `Load`'s existing blocks — verbatim, no
reordering — into four unexported functions:

```go
func decode(data []byte) (*Config, error) {
    var c Config
    if err := toml.Unmarshal(data, &c); err != nil { return nil, fmt.Errorf("parse config: %w", err) }
    if c.SchemaVersion > CurrentConfigSchemaVersion { return nil, fmt.Errorf(...upgrade homonto...) }
    return &c, nil
}
func migrate(c *Config)   { /* the [agents]->[subagents] fold, verbatim */ }
func normalize(c *Config) { /* subagent scope defaulting, verbatim */ }
func validate(c *Config) error { /* the whole validation block, verbatim, same order */ }

func Load(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil { return nil, fmt.Errorf("read config: %w", err) }
    c, err := decode(data)
    if err != nil { return nil, err }
    migrate(c)
    normalize(c)
    if err := validate(c); err != nil { return nil, err }
    return c, nil
}
```

`migrate`/`normalize` mutate `*c` (as the inline code did on the local value).
`validate` returns the first error exactly as the inline sequence did. No
validation rule is added, removed, or reordered.

## Behavior identity

Every existing config load test (valid fixtures, each validation-error case, the
agents fold, scope defaulting) pins the behavior; a pure extraction leaves them
all green. Any diff means the extraction slipped and must be fixed.

## Risk

Low — mechanical, no reordering. The config suite is the guard.

## Alternatives
- Also extract the "expand" phase (generic per-kind pipeline) — deferred; larger
  and independent of ending the Load monolith.

```

## openspec/changes/config-load-phases/tasks.md

- Source: openspec/changes/config-load-phases/tasks.md
- Lines: 1-10
- SHA256: 4d02d2178264bc706d6a2cce024ad8bdd328c7e1ab6de302cd6bbe20bb351a0f

```md
# Tasks — config-load-phases

## 1. Extract phase functions
- [ ] Extract decode/migrate/normalize/validate from config.Load in the same
      order with no behavior change; Load calls them in sequence. Config suite
      green unchanged; optionally a focused test per phase.

## 2. Verify
- [ ] `go test ./... -race`, vet, build, `openspec validate --all` green;
      config load/validation tests pass unchanged.

```

## openspec/changes/config-load-phases/specs/config-model/spec.md

- Source: openspec/changes/config-load-phases/specs/config-model/spec.md
- Lines: 1-17
- SHA256: 68eda79328f1265fcf348e7cc7fbc6cd7fd786cd4a655d308505d14fabf1b21d

```md
# config-model

## ADDED Requirements

### Requirement: Config loading runs as explicit ordered phases

Config loading SHALL run as explicit, ordered phases — decode (parse + schema
version guard), migrate (legacy-form folding), normalize (defaulting), and
validate — rather than as a single monolithic function. Each phase MUST run in
that order, and the observable result (the loaded config, and every validation
error) MUST be identical to the prior monolithic loader.

#### Scenario: Loading a config runs decode, migrate, normalize, validate in order

- **WHEN** a config is loaded
- **THEN** it is decoded, migrated, normalized, and validated in that order, and
  the resulting config and any error are identical to the prior loader

```
