package opencode

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/noviopenworks/homonto/internal/jsonutil"
	"github.com/tidwall/gjson"
)

func contains(ss []string, x string) bool { return slices.Contains(ss, x) }

func arrayHas(doc []byte, path, elem string) bool {
	for _, v := range gjson.GetBytes(doc, path).Array() {
		if v.String() == elem {
			return true
		}
	}
	return false
}

func mustJSON(v any) string { b, _ := json.Marshal(v); return string(b) }

func hasPrefix(s, p string) bool { return strings.HasPrefix(s, p) }
func trim(s, p string) string    { return strings.TrimPrefix(s, p) }

// recordedDst extracts the destination path from a skill entry's recorded
// Desired value, which is stored as "dst -> src". The recorded dst is where the
// link physically lives, independent of the adapter's current scope — so a
// pending scope switch (which changes skillsDir but not the applied link) is
// read at the right location instead of looking absent. Returns false when the
// value is not in the expected form.
func recordedDst(desired string) (string, bool) {
	dst, _, found := strings.Cut(desired, " -> ")
	return dst, found
}

// managedPrefix reports whether a state key is in a namespace this adapter
// manages — only those are eligible for pruning.
func managedPrefix(k string) bool {
	for _, p := range []string{"mcp.", "setting.", "tui.", "plugin.", "skill.", "command.", "subagent."} {
		if strings.HasPrefix(k, p) {
			return true
		}
	}
	return false
}

func readStandardized(path string) ([]byte, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return jsonutil.Standardize(nil)
	}
	if err != nil {
		return nil, err
	}
	doc, err := jsonutil.Standardize(b)
	if err != nil {
		return nil, err
	}
	if err := jsonutil.ObjectRoot(doc); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return doc, nil
}
