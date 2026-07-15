package main

import (
	"github.com/spf13/cobra"
)

func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "pulse",
		Short:         "Developer workflow orchestration tool",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	cmd.AddCommand(
		NewServeCommand(),
		NewTUICommand(),
		NewStatusCommand(),
		NewAuthCommand(),
	)
	return cmd
}
