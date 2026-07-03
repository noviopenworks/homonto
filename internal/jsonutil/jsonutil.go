package jsonutil

import (
	"github.com/tailscale/hujson"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

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

func pretty(b []byte) ([]byte, error) {
	v, err := hujson.Parse(b)
	if err != nil {
		return b, nil // already valid JSON; return as-is
	}
	v.Format()
	return v.Pack(), nil
}
