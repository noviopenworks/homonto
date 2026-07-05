package claude

import (
	"encoding/json"
	"slices"
	"strings"

	"github.com/tidwall/gjson"
)

func contains(ss []string, x string) bool { return slices.Contains(ss, x) }

func mustJSON(v any) string { b, _ := json.Marshal(v); return string(b) }

func hasPrefix(s, p string) bool { return strings.HasPrefix(s, p) }
func trim(s, p string) string    { return strings.TrimPrefix(s, p) }

// managedPrefix reports whether a state key is in a namespace this adapter
// manages — only those are eligible for pruning.
func managedPrefix(k string) bool {
	for _, p := range []string{"mcp.", "setting.", "plugin.", "skill."} {
		if strings.HasPrefix(k, p) {
			return true
		}
	}
	return false
}

// objMembers returns the immediate members of the object at root as key -> raw JSON.
func objMembers(doc []byte, root string) map[string]string {
	out := map[string]string{}
	gjson.GetBytes(doc, root).ForEach(func(k, v gjson.Result) bool {
		out[k.String()] = v.Raw
		return true
	})
	return out
}
