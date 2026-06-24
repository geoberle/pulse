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

// BindFlags registers CLI flags on the cobra command.
func (o *RawOptions) BindFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&o.ConfigFile, "config-file", o.ConfigFile, "Path to config file")
	cmd.Flags().StringVar(&o.PromptsFile, "prompts-file", o.PromptsFile, "Path to prompts file")
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

// Complete finalizes options for execution. Client construction (Jira,
// GitHub, LLM) goes here — Jira.Email and Jira.Token are optional in
// config and must be resolved (env, keychain) before constructing the
// Jira client.
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
