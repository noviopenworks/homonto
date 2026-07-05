package jsonutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tailscale/hujson"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// pathSpecials are the bytes gjson/sjson treat as path syntax inside a key:
// separators (. |), wildcards (* ?), query/modifier markers (# @), and the
// escape character itself. Verified empirically: an unescaped dot nests
// objects, and an unescaped @, | or # makes an sjson write vanish silently.
const pathSpecials = `.|*?#@\`

// EscapePath escapes one config-supplied name for use as a single segment of
// a gjson/sjson dotted path, so the segment addresses a literal key instead
// of being interpreted as path syntax. gjson and sjson honor the same
// backslash escaping, so escaped reads and writes address the same key.
func EscapePath(segment string) string {
	if !strings.ContainsAny(segment, pathSpecials) {
		return segment
	}
	var b strings.Builder
	b.Grow(len(segment) + 4)
	for i := 0; i < len(segment); i++ {
		if strings.IndexByte(pathSpecials, segment[i]) >= 0 {
			b.WriteByte('\\')
		}
		b.WriteByte(segment[i])
	}
	return b.String()
}

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

// ObjectRoot verifies the document's root value is a JSON object. Managed
// configs are always objects; a keyed write into an array or scalar root
// would silently corrupt it, so anything else is rejected up front.
func ObjectRoot(doc []byte) error {
	t := bytes.TrimSpace(doc)
	if len(t) > 0 && t[0] == '{' {
		return nil
	}
	return fmt.Errorf("root is not a JSON object (found %s)", rootKind(t))
}

// rootKind names the root value of a valid JSON document by its first byte.
func rootKind(t []byte) string {
	if len(t) == 0 {
		return "an empty document"
	}
	switch t[0] {
	case '[':
		return "an array"
	case '"':
		return "a string"
	case 't', 'f':
		return "a boolean"
	case 'n':
		return "null"
	default:
		return "a number"
	}
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
