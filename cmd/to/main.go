package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/noviopenworks/homonto/internal/tocli"
)

func main() {
	if err := tocli.NewRootCmd().Execute(); err != nil {
		// doctor --quiet's contract is exit-code-only: print nothing, so a hook
		// capturing stderr sees nothing either.
		if !errors.Is(err, tocli.ErrQuietFindings) {
			fmt.Fprintln(os.Stderr, "error:", err)
		}
		os.Exit(1)
	}
}
