package engine

import "fmt"

// catalogUpgradeFinding reports a pending catalog upgrade: when the recorded
// catalog version differs from the embedded one, apply would re-materialize the
// catalog, but until then the materialized catalog is stale. It returns the
// finding text and whether an upgrade is pending.
func catalogUpgradeFinding(recorded, embedded string) (string, bool) {
	if recorded == embedded {
		return "", false
	}
	rec := recorded
	if rec == "" {
		rec = "(none)"
	}
	return fmt.Sprintf("warn: catalog upgrade pending: recorded %s, embedded %s — run `homonto apply`", rec, embedded), true
}
