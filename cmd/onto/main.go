package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/noviopenworks/homonto/internal/ontocli"
)

func main() {
	if err := ontocli.NewRootCmd().Execute(); err != nil {
		// doctor --quiet's contract is exit-code-only: print nothing, so a hook
		// capturing stderr sees nothing either.
		if !errors.Is(err, ontocli.ErrQuietFindings) {
			fmt.Fprintln(os.Stderr, "error:", err)
		}
		os.Exit(1)
	}
}
