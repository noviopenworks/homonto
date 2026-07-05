package jsonutil

import (
	"encoding/json"
	"fmt"

	"github.com/tailscale/hujson"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// Canonical returns a key-order-independent, whitespace-normalized form of a
// JSON value (encoding/json marshals map keys sorted). Non-JSON input is
// returned unchanged, so unresolved ${...} tokens pass through untouched.
func Canonical(raw string) string {
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return raw
	}
	b, err := json.Marshal(v)
	if err != nil {
		return raw
	}
	return string(b)
}

var opts = &sjson.Options{Optimistic: false}

// SetJSON sets a dotted path to value, preserving the rest of the document.
func SetJSON(existing []byte, path string, value any) ([]byte, error) {
	if len(existing) == 0 {
		existing = []byte("{}")
	}
	out, err := sjson.SetBytesOptions(existing, path, value, opts)
	if err != nil {
		return nil, err
	}
	return pretty(out)
}

// DeleteJSON removes the value at a dotted path, preserving the rest of the
// document. A missing path is not an error — the delete is already done.
func DeleteJSON(existing []byte, path string) ([]byte, error) {
	if len(existing) == 0 {
		return []byte("{}"), nil
	}
	out, err := sjson.DeleteBytes(existing, path)
	if err != nil {
		return nil, err
	}
	return pretty(out)
}

// GetJSON returns the raw JSON of the value at path and whether it exists.
func GetJSON(existing []byte, path string) (string, bool) {
	r := gjson.GetBytes(existing, path)
	if !r.Exists() {
		return "", false
	}
	return r.Raw, true
}

// Standardize converts JSONC to plain JSON (dropping comments). Empty -> "{}".
func Standardize(jsonc []byte) ([]byte, error) {
	if len(jsonc) == 0 {
		return []byte("{}"), nil
	}
	v, err := hujson.Parse(jsonc)
	if err != nil {
		return nil, err
	}
	v.Standardize()
	return v.Pack(), nil
}

// EnsureArrayElem appends string elem to the array at path if absent.
func EnsureArrayElem(existing []byte, path, elem string) ([]byte, error) {
	for _, v := range gjson.GetBytes(existing, path).Array() {
		if v.String() == elem {
			return existing, nil
		}
	}
	out, err := sjson.SetBytesOptions(existing, path+".-1", elem, opts)
	if err != nil {
		return nil, err
	}
	return pretty(out)
}

// RemoveArrayElem deletes every occurrence of string elem from the array at
// path; an absent elem (or array) leaves the document untouched.
func RemoveArrayElem(existing []byte, path, elem string) ([]byte, error) {
	arr := gjson.GetBytes(existing, path).Array()
	out, changed := existing, false
	// Walk backwards so earlier indexes stay valid after each removal.
	for i := len(arr) - 1; i >= 0; i-- {
		if arr[i].String() != elem {
			continue
		}
		var err error
		out, err = sjson.DeleteBytes(out, fmt.Sprintf("%s.%d", path, i))
		if err != nil {
			return nil, err
		}
		changed = true
	}
	if !changed {
		return existing, nil
	}
	return pretty(out)
}

func pretty(b []byte) ([]byte, error) {
	v, err := hujson.Parse(b)
	if err != nil {
		return b, nil // already valid JSON; return as-is
	}
	v.Format()
	return v.Pack(), nil
}
