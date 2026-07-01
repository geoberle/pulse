package main

import (
	"context"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"github.com/go-logr/logr/funcr"
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
	log := newLogger()
	ctx = logr.NewContext(ctx, log)

	validated, err := opts.Validate()
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	completed, err := validated.Complete(ctx)
	if err != nil {
		return fmt.Errorf("completion failed: %w", err)
	}
	return completed.Run(ctx)
}

func newLogger() logr.Logger {
	return funcr.New(func(prefix, args string) {
		if len(prefix) > 0 {
			fmt.Fprintf(os.Stderr, "%s %s\n", prefix, args)
		} else {
			fmt.Fprintf(os.Stderr, "%s\n", args)
		}
	}, funcr.Options{Verbosity: 1})
}
