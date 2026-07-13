package claude

import (
	"encoding/json"
	"slices"
	"strings"

	"github.com/noviopenworks/homonto/internal/adapter"
)

func contains(ss []string, x string) bool { return slices.Contains(ss, x) }

func mustJSON(v any) string { b, _ := json.Marshal(v); return string(b) }

func hasPrefix(s, p string) bool { return strings.HasPrefix(s, p) }
func trim(s, p string) string    { return strings.TrimPrefix(s, p) }

// filePrefix reports whether a state key is a file-projection namespace pruned
// by the generic delete loop. The structured-document prefixes (mcp./setting./
// plugin./pluginconfig./marketplace.) are pruned by their structproj.Project
// calls instead, so they are excluded here to avoid a double delete.
func filePrefix(k string) bool {
	for _, p := range []string{"skill.", "command.", "subagent."} {
		if strings.HasPrefix(k, p) {
			return true
		}
	}
	return false
}

// filterDesired returns the subset of desired values whose keys are in prefix,
// so each structproj namespace sees only the keys it owns.
func filterDesired(m map[string]string, prefix string) map[string]string {
	out := map[string]string{}
	for k, v := range m {
		if strings.HasPrefix(k, prefix) {
			out[k] = v
		}
	}
	return out
}

// filterChanges returns the subset of changes whose keys are in prefix, so each
// structproj namespace applies only the changes it owns.
func filterChanges(changes []adapter.Change, prefix string) []adapter.Change {
	var out []adapter.Change
	for _, c := range changes {
		if strings.HasPrefix(c.Key, prefix) {
			out = append(out, c)
		}
	}
	return out
}
