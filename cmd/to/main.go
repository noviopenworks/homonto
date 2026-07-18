package main

import (
	"fmt"
	"os"

	"github.com/noviopenworks/homonto/internal/tocli"
)

func main() {
	if err := tocli.NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
