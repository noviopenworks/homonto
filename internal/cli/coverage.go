package cli

import (
	"fmt"
	"strings"
)

// coverageComplete reports incomplete coverage: it returns a non-nil error when
// any adapter warning was emitted during the run, so plan/status/apply never
// print a clean conclusion (or exit zero) after a skipped or degraded adapter
// (F45). The warnings themselves are printed separately by the caller.
func coverageComplete(warnings []string) error {
	if len(warnings) == 0 {
		return nil
	}
	return fmt.Errorf("coverage was incomplete (one or more adapters skipped or degraded): %s", strings.Join(warnings, "; "))
}
