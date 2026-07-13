package adapter

import (
	"fmt"

	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/secret"
	"github.com/noviopenworks/homonto/internal/state"
)

// SecretRedaction is placed in Change.Old for any change on a secret-bearing
// key, so plan output and logs never contain a resolved secret value.
const SecretRedaction = "«secret»"

// Action is a planned operation's kind. It is a defined type so an unknown
// action is a compile- or validation-time error rather than a silent no-op; the
// constants keep the historical string values, so existing string-literal
// construction and comparison stay valid.
type Action string

const (
	ActionCreate Action = "create"
	ActionUpdate Action = "update"
	ActionDelete Action = "delete"
	ActionNoop   Action = "noop"
	ActionAdopt  Action = "adopt"
)

// Valid reports whether a is one of the defined operations.
func (a Action) Valid() bool {
	switch a {
	case ActionCreate, ActionUpdate, ActionDelete, ActionNoop, ActionAdopt:
		return true
	default:
		return false
	}
}

// Change is a single planned modification. Old/New hold JSON-encoded values.
// New may contain unresolved ${...} secret tokens; for secret-bearing keys Old
// is redacted to SecretRedaction and never carries the on-disk resolved value.
// Deletes (a key in state but no longer declared) always redact Old — a
// removed key's provenance is stale by definition — and carry no New.
type Change struct {
	Action Action
	Key    string
	Old    string
	New    string
}

// ChangeSet is one tool's planned changes.
type ChangeSet struct {
	Tool    string
	Changes []Change
}

// Validate fails closed on a change set that must never reach apply: a tool not
// backed by a registered adapter (today silently skipped), or an operation whose
// action is not one of the defined operations (today a silent no-op). Legal
// plans — every operation a registered adapter emits — always pass.
func (cs ChangeSet) Validate(knownTools map[string]bool) error {
	if !knownTools[cs.Tool] {
		return fmt.Errorf("plan validation: unknown tool %q (not a registered adapter)", cs.Tool)
	}
	for _, c := range cs.Changes {
		if !c.Action.Valid() {
			return fmt.Errorf("plan validation: unknown action %q for key %q (tool %q)", c.Action, c.Key, cs.Tool)
		}
	}
	return nil
}

// Adapter projects desired config into one target tool.
type Adapter interface {
	Name() string
	Plan(c *config.Config, st *state.State) (ChangeSet, error)
	Apply(cs ChangeSet, res *secret.Resolver, st *state.State) error
	// ObserveHashes returns, for each state-recorded key of this tool still
	// present on disk, a hash of the CURRENT on-disk value computed the same way
	// the key's Entry.Applied was stored at apply — so an unchanged key hashes
	// back to its Entry.Applied. Recorded keys absent from disk are omitted (the
	// engine infers "missing"). Only hashes escape the adapter: raw on-disk
	// values (which may include resolved secrets) never leave it.
	ObserveHashes(st *state.State) (map[string]string, error)
}
