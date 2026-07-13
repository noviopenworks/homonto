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

// managedPrefix reports whether a state key is pruned by the generic delete
// loop: plugin.* (bespoke array membership) and the file-projection prefixes.
// The structured-document prefixes (mcp./setting./tui.) are pruned by their
// structproj.Project calls instead, so they are excluded here to avoid a double
// delete.
func managedPrefix(k string) bool {
	for _, p := range []string{"plugin.", "skill.", "command.", "subagent."} {
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
