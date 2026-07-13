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
