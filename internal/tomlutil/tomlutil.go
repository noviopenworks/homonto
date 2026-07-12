// Package tomlutil is a TOML codec for the adapter projection core: it reads,
// sets, and deletes a value at a dotted key path in a TOML document while
// preserving unmanaged tables and keys, bridging TOML values to canonical JSON
// so state hashing stays format-independent. Comment preservation is a
// non-goal (a write normalizes the document), matching OpenCode's JSONC limit.
package tomlutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
)

// EnsureRoot normalizes an empty or whitespace-only document to an empty TOML
// object so a first Set has a root to write into.
func EnsureRoot(doc []byte) ([]byte, error) {
	if len(bytes.TrimSpace(doc)) == 0 {
		return []byte{}, nil
	}
	var m map[string]any
	if err := toml.Unmarshal(doc, &m); err != nil {
		return nil, fmt.Errorf("tomlutil: parse: %w", err)
	}
	return doc, nil
}

// Get returns the value at a dotted path encoded as canonical JSON, and whether
// it is present.
func Get(doc []byte, path string) (string, bool) {
	m, err := load(doc)
	if err != nil {
		return "", false
	}
	v, ok := getPath(m, splitPath(path))
	if !ok {
		return "", false
	}
	enc, err := json.Marshal(v)
	if err != nil {
		return "", false
	}
	return Canonical(string(enc)), true
}

// Set assigns jsonValue (a JSON-encoded value) at a dotted path, creating
// intermediate tables and preserving unmanaged content.
func Set(doc []byte, path, jsonValue string) ([]byte, error) {
	m, err := load(doc)
	if err != nil {
		return nil, err
	}
	var v any
	if err := json.Unmarshal([]byte(jsonValue), &v); err != nil {
		return nil, fmt.Errorf("tomlutil: decode value: %w", err)
	}
	segs := splitPath(path)
	if len(segs) == 0 {
		return nil, fmt.Errorf("tomlutil: empty path")
	}
	setPath(m, segs, v)
	return marshal(m)
}

// Delete removes the value at a dotted path and prunes any parent tables it
// leaves empty.
func Delete(doc []byte, path string) ([]byte, error) {
	m, err := load(doc)
	if err != nil {
		return nil, err
	}
	deletePath(m, splitPath(path))
	return marshal(m)
}

// Canonical renders a JSON-encoded value in a stable, key-sorted form so equal
// values hash identically regardless of key order or spacing.
func Canonical(jsonValue string) string {
	var v any
	if err := json.Unmarshal([]byte(jsonValue), &v); err != nil {
		return jsonValue
	}
	out, err := json.Marshal(v) // Go sorts map keys on marshal
	if err != nil {
		return jsonValue
	}
	return string(out)
}

func load(doc []byte) (map[string]any, error) {
	m := map[string]any{}
	if len(bytes.TrimSpace(doc)) == 0 {
		return m, nil
	}
	if err := toml.Unmarshal(doc, &m); err != nil {
		return nil, fmt.Errorf("tomlutil: parse: %w", err)
	}
	return m, nil
}

func marshal(m map[string]any) ([]byte, error) {
	var buf bytes.Buffer
	enc := toml.NewEncoder(&buf)
	enc.SetIndentTables(true)
	if err := enc.Encode(m); err != nil {
		return nil, fmt.Errorf("tomlutil: encode: %w", err)
	}
	return buf.Bytes(), nil
}

func splitPath(path string) []string {
	if path == "" {
		return nil
	}
	return strings.Split(path, ".")
}

func getPath(m map[string]any, segs []string) (any, bool) {
	cur := any(m)
	for _, s := range segs {
		mm, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		cur, ok = mm[s]
		if !ok {
			return nil, false
		}
	}
	return cur, true
}

func setPath(m map[string]any, segs []string, v any) {
	cur := m
	for _, s := range segs[:len(segs)-1] {
		next, ok := cur[s].(map[string]any)
		if !ok {
			next = map[string]any{}
			cur[s] = next
		}
		cur = next
	}
	cur[segs[len(segs)-1]] = v
}

// deletePath removes the leaf and prunes emptied ancestor tables.
func deletePath(m map[string]any, segs []string) {
	if len(segs) == 0 {
		return
	}
	// Walk to the parent, tracking the chain so we can prune empties.
	chain := []map[string]any{m}
	cur := m
	for _, s := range segs[:len(segs)-1] {
		next, ok := cur[s].(map[string]any)
		if !ok {
			return // path does not exist; nothing to delete
		}
		chain = append(chain, next)
		cur = next
	}
	delete(cur, segs[len(segs)-1])
	// Prune empty parents from the leaf upward (but never the root map).
	for i := len(chain) - 1; i >= 1; i-- {
		if len(chain[i]) == 0 {
			delete(chain[i-1], segs[i-1])
		} else {
			break
		}
	}
}
