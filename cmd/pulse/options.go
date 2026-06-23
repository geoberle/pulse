package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/geoberle/pulse/internal/config"
)

// RawOptions holds unvalidated CLI flag values. Populated by cobra flag
// binding before any validation occurs.
type RawOptions struct {
	// ConfigFile is the path to the application config YAML.
	ConfigFile string

	// PromptsFile is the path to the prompt templates YAML.
	PromptsFile string
}

type validatedOptions struct {
	Config  *config.Config
	Prompts *config.Prompts
}

// ValidatedOptions wraps validated configuration that has passed all
// structural checks. The unexported inner struct prevents external
// callers from constructing one without going through Validate().
type ValidatedOptions struct {
	validatedOptions
}

type completedOptions struct {
	Config  *config.Config
	Prompts *config.Prompts
}

// Options holds fully completed configuration ready for execution.
// The unexported inner struct prevents external callers from
// constructing one without going through Complete().
type Options struct {
	completedOptions
}

// DefaultOptions returns RawOptions populated with XDG-compliant default paths.
func DefaultOptions() (*RawOptions, error) {
	cfgPath, err := config.DefaultConfigPath()
	if err != nil {
		return nil, err
	}
	promptsPath, err := config.DefaultPromptsPath()
	if err != nil {
		return nil, err
	}
	return &RawOptions{
		ConfigFile:  cfgPath,
		PromptsFile: promptsPath,
	}, nil
}

// BindOptions registers CLI flags for the given RawOptions on the cobra command.
func BindOptions(opts *RawOptions, cmd *cobra.Command) error {
	cmd.Flags().StringVar(&opts.ConfigFile, "config-file", opts.ConfigFile, "Path to config file")
	cmd.Flags().StringVar(&opts.PromptsFile, "prompts-file", opts.PromptsFile, "Path to prompts file")
	return nil
}

// Validate loads config and prompts from disk and checks all structural
// invariants: required fields, valid durations, HTTPS host, and template
// syntax. Returns an error at startup rather than failing mid-session.
func (o *RawOptions) Validate() (*ValidatedOptions, error) {
	cfg, err := config.LoadConfig(o.ConfigFile)
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}

	prompts, err := config.LoadPrompts(o.PromptsFile)
	if err != nil {
		return nil, fmt.Errorf("prompts: %w", err)
	}
	if err := prompts.ValidateTemplates(); err != nil {
		return nil, fmt.Errorf("prompts: %w", err)
	}

	return &ValidatedOptions{
		validatedOptions: validatedOptions{
			Config:  cfg,
			Prompts: prompts,
		},
	}, nil
}

// Complete finalizes options for execution. Drops raw inputs and retains
// only the validated configuration needed at runtime.
func (o *ValidatedOptions) Complete() (*Options, error) {
	return &Options{
		completedOptions: completedOptions{
			Config:  o.Config,
			Prompts: o.Prompts,
		},
	}, nil
}

// Run executes the main application loop.
func (o *Options) Run(_ context.Context) error {
	fmt.Printf("Pulse running (poll every %s, %d repos, project %s)\n",
		o.Config.PollInterval,
		len(o.Config.Repos),
		o.Config.JiraProject,
	)
	return nil
}
