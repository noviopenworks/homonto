// Package structproj is the adapter contract: a format-agnostic managed-key
// projection engine. It owns the plan/apply/observe control flow that Claude and
// OpenCode otherwise each re-implement — diffing desired values against disk and
// recorded state to emit create/update/delete/noop/adopt changes, writing only
// managed keys while preserving unmanaged content, and re-hashing recorded keys
// for drift. A new adapter supplies only a Codec (its file format), a state-key
// prefix, and a key→document-path mapping.
package structproj

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/noviopenworks/homonto/internal/adapter"
	"github.com/noviopenworks/homonto/internal/secret"
	"github.com/noviopenworks/homonto/internal/state"
)

// Codec abstracts a structured config document of some format (JSON, TOML, …).
// Values crossing this boundary are JSON-encoded strings so state hashing stays
// format-independent.
type Codec interface {
	// EnsureRoot normalizes an empty/whitespace document to a writable root.
	EnsureRoot(doc []byte) ([]byte, error)
	// Get returns the canonical JSON value at a path and whether it is present.
	Get(doc []byte, path string) (string, bool)
	// Set assigns a JSON-encoded value at a path, preserving unmanaged content.
	Set(doc []byte, path, jsonValue string) ([]byte, error)
	// Delete removes the value at a path, pruning parents it empties.
	Delete(doc []byte, path string) ([]byte, error)
	// Canonical renders a JSON-encoded value in a stable form for compare/hash.
	Canonical(jsonValue string) string
}

// PathFor maps a full state key (prefix+name) to the codec document path.
type PathFor func(stateKey string) string

// Project diffs desired (state-key → JSON value) against the on-disk document
// and recorded state, producing the change list for one managed namespace. It
// reproduces the built-in adapters' semantics exactly, including secret-safe
// redaction of Old.
func Project(tool, prefix string, desired map[string]string, disk []byte, st *state.State, codec Codec, pathFor PathFor) []adapter.Change {
	var changes []adapter.Change
	declared := make(map[string]bool, len(desired))
	for key, want := range desired {
		declared[key] = true
		diskVal, hasDisk := codec.Get(disk, pathFor(key))
		e, inState := st.Get(tool, key)
		switch {
		case !hasDisk:
			changes = append(changes, adapter.Change{Action: "create", Key: key, New: want})
		case !secret.ContainsRef(want):
			if diskVal == codec.Canonical(want) {
				if inState && e.Applied == secret.Hash(diskVal) {
					changes = append(changes, adapter.Change{Action: "noop", Key: key})
				} else {
					changes = append(changes, adapter.Change{Action: "adopt", Key: key, New: want})
				}
			} else {
				old := diskVal
				// Never expose an on-disk value of unknown or secret provenance.
				if !inState || secret.ContainsRef(e.Desired) {
					old = adapter.SecretRedaction
				}
				changes = append(changes, adapter.Change{Action: "update", Key: key, Old: old, New: want})
			}
		default: // secret-bearing desired value: never read/expose the on-disk value
			if inState && e.Desired == want && e.Applied == secret.Hash(diskVal) {
				changes = append(changes, adapter.Change{Action: "noop", Key: key})
			} else {
				changes = append(changes, adapter.Change{Action: "update", Key: key, Old: adapter.SecretRedaction, New: want})
			}
		}
	}
	// Delete recorded keys in this namespace no longer declared.
	for _, k := range st.Keys(tool) {
		if !strings.HasPrefix(k, prefix) || declared[k] {
			continue
		}
		changes = append(changes, adapter.Change{Action: "delete", Key: k, Old: adapter.SecretRedaction})
	}
	sort.SliceStable(changes, func(i, j int) bool { return changes[i].Key < changes[j].Key })
	return changes
}

// Apply writes the changes into the document (managed keys only), records state,
// and reports whether the document changed. noop/adopt are state-only and leave
// the document byte-for-byte untouched. Secrets are resolved via res.
func Apply(tool, prefix string, changes []adapter.Change, disk []byte, codec Codec, res *secret.Resolver, st *state.State, pathFor PathFor) ([]byte, bool, error) {
	doc := disk
	changed := false
	for _, c := range changes {
		switch c.Action {
		case "noop":
			continue
		case "adopt":
			val, err := res.ResolveJSON(c.New)
			if err != nil {
				return nil, false, err
			}
			st.Set(tool, c.Key, c.New, secret.Hash(codec.Canonical(mustJSON(val))))
		case "delete":
			var err error
			if doc, err = codec.EnsureRoot(doc); err != nil {
				return nil, false, err
			}
			if doc, err = codec.Delete(doc, pathFor(c.Key)); err != nil {
				return nil, false, err
			}
			changed = true
			st.Delete(tool, c.Key)
		default: // create | update
			val, err := res.ResolveJSON(c.New)
			if err != nil {
				return nil, false, err
			}
			if doc, err = codec.EnsureRoot(doc); err != nil {
				return nil, false, err
			}
			if doc, err = codec.Set(doc, pathFor(c.Key), mustJSON(val)); err != nil {
				return nil, false, err
			}
			changed = true
			st.Set(tool, c.Key, c.New, secret.Hash(codec.Canonical(mustJSON(val))))
		}
	}
	return doc, changed, nil
}

// Observe re-hashes each recorded key of this namespace still present on disk,
// the same way Apply stored Entry.Applied, so an unchanged key hashes back to
// its recorded value. Keys absent from disk are omitted.
func Observe(tool, prefix string, disk []byte, st *state.State, codec Codec, pathFor PathFor) map[string]string {
	out := map[string]string{}
	for _, k := range st.Keys(tool) {
		if !strings.HasPrefix(k, prefix) {
			continue
		}
		if v, ok := codec.Get(disk, pathFor(k)); ok {
			out[k] = secret.Hash(v)
		}
	}
	return out
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "null"
	}
	return string(b)
}
