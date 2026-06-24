package main

import (
	"fmt"
	"os"
)

// TODO: Wire signal handling (SIGINT/SIGTERM) to context cancellation
// for graceful TUI shutdown.
func main() {
	cmd, err := NewCommand()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
