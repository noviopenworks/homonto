# Secrets — referenced, never stored

Never put a plaintext secret in `homonto.toml`. Secret values live outside the
repo and are referenced by token; homonto keeps the value unresolved everywhere
except the moment of writing, and never stores what it resolved.

## The two reference forms

```toml
[mcps.brave]
command = ["npx", "-y", "@modelcontextprotocol/server-brave-search"]
env = { BRAVE_API_KEY = "${pass:ai/brave}" }   # pass-store reference

[mcps.other]
command = ["other-server"]
env = { OTHER_KEY = "${OTHER_KEY}" }           # environment reference
```

| Form | Resolved via | Failure mode |
|---|---|---|
| `${pass:path/to/secret}` | [`pass`](https://www.passwordstore.org/) — the standard Unix password store | error if `pass` is not on `PATH` or the entry is missing |
| `${ENV_VAR}` | the process environment (zero-dependency fallback) | error if the variable is unset at apply time |

`homonto doctor` flags a missing `pass`. The scaffolded `.env.example` from
`homonto init` is a natural place to document which `${ENV_VAR}` references a
config expects.

## The guarantees

- **`plan` never resolves a secret.** It neither contacts a backend nor prints
  a value — the diff shows the `${…}` token.
- **`apply` resolves all secrets up front**, only *after* you confirm, and
  **before any file is written**. One missing reference aborts the entire apply
  with nothing touched.
- **State stores a hash, never a value.** `.homonto/state.json` records the
  unresolved token plus a **sha256 hash** of the applied value. That makes it:
  - safe to share (no plaintext ever lands in state);
  - idempotent — a repeat `apply` of an unchanged secret-backed value is a
    no-op;
  - drift-aware — an out-of-band change to the resolved value on disk is still
    detected, because the hash no longer matches.
- **Adoption never short-circuits secrets.** Resources whose values contain
  secret references are always re-applied so the resolved value is verified —
  they are never silently adopted from disk (see
  [projection & state](projection-and-state.md)).

## Caveats

- The **resolved** value does end up in the tool's own config file (that is the
  point — the tool needs it). The guarantee is about `homonto.toml` and
  `.homonto/state.json`, which are the files you commit and share.
- `import` redacts values that *look* like secrets into `${pass:…}` references
  on a best-effort basis; review its output before committing.
