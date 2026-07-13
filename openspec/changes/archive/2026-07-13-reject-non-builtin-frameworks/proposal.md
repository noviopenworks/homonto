## Why
ROADMAP E1 / finding F35: `[frameworks.<name>]` accepts a `local:` source at load
(`validateResources` → `validateSource` allow builtin:/local:), but framework
expansion only acts on `builtin:` sources — so a `local:` (or otherwise
non-builtin) framework passes validation and then silently installs NOTHING. An
unsupported source/kind combination must fail loudly at load, never be a silent
no-op.
## What Changes
- Config load rejects a `[frameworks.<name>]` whose source is not `builtin:` with
  a clear error (only builtin frameworks are supported today).
## Impact
- **Code:** `internal/config/config.go` (framework validation) + test.
- **Spec:** `framework-expansion` delta (non-builtin framework source is a load error).
- **Out of scope:** the full E1 ecosystem model (versioned manifests, local/custom framework resolution).
