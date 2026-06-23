package main

import (
	"context"
	"fmt"

	"github.com/geoberle/pulse/internal/config"
	"github.com/spf13/cobra"
)

type RawOptions struct {
	ConfigFile  string
	PromptsFile string
}

type validatedOptions struct {
	*RawOptions
	Config  *config.Config
	Prompts *config.Prompts
}

type ValidatedOptions struct {
	*validatedOptions
}

type completedOptions struct {
	Config  *config.Config
	Prompts *config.Prompts
}

type Options struct {
	*completedOptions
}

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

func BindOptions(opts *RawOptions, cmd *cobra.Command) error {
	cmd.Flags().StringVar(&opts.ConfigFile, "config-file", opts.ConfigFile, "Path to config file")
	cmd.Flags().StringVar(&opts.PromptsFile, "prompts-file", opts.PromptsFile, "Path to prompts file")
	return nil
}

func (o *RawOptions) Validate(_ context.Context) (*ValidatedOptions, error) {
	cfg, err := config.LoadConfig(o.ConfigFile)
	if err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	if len(cfg.Repos) == 0 {
		return nil, fmt.Errorf("no repos configured")
	}
	if len(cfg.JiraProject) == 0 {
		return nil, fmt.Errorf("jira_project is required")
	}
	if _, err := cfg.PollDuration(); err != nil {
		return nil, fmt.Errorf("invalid poll_interval: %w", err)
	}

	prompts, err := config.LoadPrompts(o.PromptsFile)
	if err != nil {
		return nil, fmt.Errorf("prompts validation failed: %w", err)
	}

	return &ValidatedOptions{
		validatedOptions: &validatedOptions{
			RawOptions: o,
			Config:     cfg,
			Prompts:    prompts,
		},
	}, nil
}

func (o *ValidatedOptions) Complete() (*Options, error) {
	return &Options{
		completedOptions: &completedOptions{
			Config:  o.Config,
			Prompts: o.Prompts,
		},
	}, nil
}

func (o *Options) Run(_ context.Context) error {
	fmt.Printf("Pulse running (poll every %s, %d repos, project %s)\n",
		o.Config.PollInterval,
		len(o.Config.Repos),
		o.Config.JiraProject,
	)
	return nil
}
