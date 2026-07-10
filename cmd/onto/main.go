package main

import (
	"fmt"
	"os"

	"github.com/noviopenworks/homonto/internal/ontocli"
)

func main() {
	if err := ontocli.NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
