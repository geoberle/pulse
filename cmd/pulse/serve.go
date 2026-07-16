package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/go-logr/logr/funcr"
	"github.com/spf13/cobra"

	"github.com/geoberle/pulse/internal/config"
	"github.com/geoberle/pulse/internal/storage"
)

type ServeRawOptions struct {
	ConfigFile string
}

type ServeValidatedOptions struct {
	Config *config.Config
}

type ServeOptions struct {
	Config *config.Config
	Store  *storage.Store
	Log    logr.Logger
}

func NewServeCommand() *cobra.Command {
	opts, err := defaultServeOptions()
	if err != nil {
		panic(fmt.Sprintf("default serve options: %v", err))
	}
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the Pulse daemon",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServe(cmd.Context(), opts)
		},
	}
	opts.BindFlags(cmd)
	return cmd
}

func defaultServeOptions() (*ServeRawOptions, error) {
	cfgPath, err := config.DefaultConfigPath()
	if err != nil {
		return nil, err
	}
	return &ServeRawOptions{ConfigFile: cfgPath}, nil
}

func (o *ServeRawOptions) BindFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&o.ConfigFile, "config-file", o.ConfigFile, "Path to config file")
}

func (o *ServeRawOptions) Validate() (*ServeValidatedOptions, error) {
	cfg, err := config.LoadConfig(o.ConfigFile)
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	return &ServeValidatedOptions{Config: cfg}, nil
}

func (o *ServeValidatedOptions) Complete(ctx context.Context) (*ServeOptions, error) {
	log := funcr.New(func(prefix, args string) {
		if len(prefix) > 0 {
			fmt.Fprintf(os.Stderr, "%s %s\n", prefix, args)
		} else {
			fmt.Fprintf(os.Stderr, "%s\n", args)
		}
	}, funcr.Options{
		Verbosity:    1,
		LogTimestamp: true,
	})

	stateDir, err := config.DefaultStateDir()
	if err != nil {
		return nil, fmt.Errorf("state directory: %w", err)
	}
	dbPath := filepath.Join(stateDir, "pulse.db")

	store, err := storage.New(ctx, dbPath)
	if err != nil {
		return nil, fmt.Errorf("storage: %w", err)
	}

	return &ServeOptions{
		Config: o.Config,
		Store:  store,
		Log:    log,
	}, nil
}

func (o *ServeOptions) Run(ctx context.Context) error {
	defer func() {
		if err := o.Store.Close(); err != nil {
			o.Log.Error(err, "closing store")
		}
	}()

	o.Log.Info("server starting",
		"repos", len(o.Config.Repos),
		"jira_project", o.Config.Jira.Project,
	)
	<-ctx.Done()
	o.Log.Info("server stopping")
	return nil
}

func runServe(ctx context.Context, opts *ServeRawOptions) error {
	validated, err := opts.Validate()
	if err != nil {
		return err
	}
	completed, err := validated.Complete(ctx)
	if err != nil {
		return err
	}
	return completed.Run(ctx)
}
