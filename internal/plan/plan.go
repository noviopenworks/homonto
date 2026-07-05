package plan

import (
	"fmt"
	"strings"

	"github.com/noviopenworks/homonto/internal/adapter"
)

// HasChanges reports whether any change is not a noop.
func HasChanges(sets []adapter.ChangeSet) bool {
	for _, s := range sets {
		for _, c := range s.Changes {
			if c.Action != "noop" {
				return true
			}
		}
	}
	return false
}

// HasAdoptions reports whether any change is an adopt.
func HasAdoptions(sets []adapter.ChangeSet) bool {
	for _, s := range sets {
		for _, c := range s.Changes {
			if c.Action == "adopt" {
				return true
			}
		}
	}
	return false
}

// Render produces a terraform-style plan. It never resolves secrets: values are
// printed verbatim (New carries unresolved tokens; secret Old is pre-redacted).
func Render(sets []adapter.ChangeSet) string {
	var b strings.Builder
	for _, s := range sets {
		var lines []string
		for _, c := range s.Changes {
			switch c.Action {
			case "create":
				lines = append(lines, fmt.Sprintf("  + %s = %s", c.Key, c.New))
			case "update":
				lines = append(lines, fmt.Sprintf("  ~ %s: %s -> %s", c.Key, c.Old, c.New))
			case "delete":
				lines = append(lines, fmt.Sprintf("  - %s", c.Key))
			}
		}
		if len(lines) == 0 {
			continue
		}
		fmt.Fprintf(&b, "%s:\n%s\n", s.Tool, strings.Join(lines, "\n"))
	}
	return b.String()
}
