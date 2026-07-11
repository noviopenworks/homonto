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
	for _, p := range []string{"mcp.", "setting.", "plugin.", "pluginconfig.", "marketplace.", "skill.", "command.", "subagent."} {
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
