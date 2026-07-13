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
