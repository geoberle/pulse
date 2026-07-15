package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewTUICommand() *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Connect TUI to running Pulse daemon",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintln(cmd.OutOrStderr(), "tui: not implemented")
			return err
		},
	}
}
