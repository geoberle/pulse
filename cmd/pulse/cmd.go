package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func NewCommand() (*cobra.Command, error) {
	opts, err := DefaultOptions()
	if err != nil {
		return nil, err
	}
	cmd := &cobra.Command{
		Use:           "pulse",
		Short:         "Developer workflow dashboard",
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd.Context(), opts)
		},
	}
	opts.BindFlags(cmd)
	return cmd, nil
}

func run(ctx context.Context, opts *RawOptions) error {
	validated, err := opts.Validate()
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	completed, err := validated.Complete()
	if err != nil {
		return fmt.Errorf("completion failed: %w", err)
	}
	return completed.Run(ctx)
}
