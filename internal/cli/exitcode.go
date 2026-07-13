package cli

import (
	"fmt"
	"os"
)

// exitCode is the taxonomy exit code a command set via --exit-code; Execute
// returns it. It is reset at the start of every Execute. Single-run CLI, so a
// package-level sink is safe.
var exitCode int

func setExitCode(c int) { exitCode = c }

// Execute runs the root command with the given args and returns the process
// exit code: 1 on error, otherwise the taxonomy code a command set under
// --exit-code (0 by default). main() passes the result to os.Exit.
func Execute(args []string) int {
	exitCode = 0
	root := NewRootCmd()
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	return exitCode
}

// planExitCode maps plan state to the opt-in taxonomy: 2 when there are pending
// changes or remote repins, else 0.
func planExitCode(hasChanges bool, repins int) int {
	if hasChanges || repins > 0 {
		return 2
	}
	return 0
}

// statusExitCode maps status state to the opt-in taxonomy: 3 when drift is
// present, 2 when only pending config changes, else 0.
func statusExitCode(hasDrift bool, pending int) int {
	switch {
	case hasDrift:
		return 3
	case pending > 0:
		return 2
	default:
		return 0
	}
}
