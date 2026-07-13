package main

import (
	"os"

	"github.com/noviopenworks/homonto/internal/cli"
)

func main() {
	os.Exit(cli.Execute(os.Args[1:]))
}
