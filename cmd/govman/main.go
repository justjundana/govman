package main

import (
	"os"

	_cli "github.com/justjundana/govman/internal/cli"
)

// main is the entry point for the Govman CLI.
// It runs cli.Execute and exits with a non-zero status code if an error occurs.
func main() {
	if err := _cli.Execute(); err != nil {
		os.Exit(1)
	}
}
