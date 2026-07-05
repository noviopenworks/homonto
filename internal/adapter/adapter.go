package adapter

import (
	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/secret"
	"github.com/noviopenworks/homonto/internal/state"
)

// SecretRedaction is placed in Change.Old for any change on a secret-bearing
// key, so plan output and logs never contain a resolved secret value.
const SecretRedaction = "«secret»"

// Change is a single planned modification. Old/New hold JSON-encoded values.
// New may contain unresolved ${...} secret tokens; for secret-bearing keys Old
// is redacted to SecretRedaction and never carries the on-disk resolved value.
// Deletes (a key in state but no longer declared) always redact Old — a
// removed key's provenance is stale by definition — and carry no New.
type Change struct {
	Action string // "create" | "update" | "delete" | "noop" | "adopt"
	Key    string
	Old    string
	New    string
}

// ChangeSet is one tool's planned changes.
type ChangeSet struct {
	Tool    string
	Changes []Change
}

// Adapter projects desired config into one target tool.
type Adapter interface {
	Name() string
	Plan(c *config.Config, st *state.State) (ChangeSet, error)
	Apply(cs ChangeSet, res *secret.Resolver, st *state.State) error
}
