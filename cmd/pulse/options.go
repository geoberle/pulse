package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-logr/logr/funcr"
	gogithub "github.com/google/go-github/v72/github"
	"github.com/spf13/cobra"

	"github.com/geoberle/pulse/internal/config"
	"github.com/geoberle/pulse/internal/engine"
	loghandler "github.com/geoberle/pulse/internal/handler/log"
	"github.com/geoberle/pulse/internal/informer"
	"github.com/geoberle/pulse/internal/poller"
	ghpoller "github.com/geoberle/pulse/internal/poller/github"
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
	Pollers []poller.Poller
	User    string
	PollDur time.Duration
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

// Complete constructs external clients and finalizes options for execution.
func (o *ValidatedOptions) Complete(ctx context.Context) (*Options, error) {
	pollDur, err := o.Config.PollDuration()
	if err != nil {
		return nil, fmt.Errorf("poll interval: %w", err)
	}

	ghToken, err := ghpoller.Token(ctx)
	if err != nil {
		return nil, fmt.Errorf("github auth: %w", err)
	}
	ghUser, err := ghpoller.User(ctx)
	if err != nil {
		return nil, fmt.Errorf("github user: %w", err)
	}

	ghClient := gogithub.NewClient(nil).WithAuthToken(ghToken)
	ghPoll := ghpoller.NewPoller(ghClient.PullRequests, ghClient.Checks, o.Config.Repos, ghUser)

	return &Options{
		completedOptions: completedOptions{
			Config:  o.Config,
			Prompts: o.Prompts,
			Pollers: []poller.Poller{ghPoll},
			User:    ghUser,
			PollDur: pollDur,
		},
	}, nil
}

// Run executes the main application loop.
func (o *Options) Run(ctx context.Context) error {
	log := newLogger()
	// 5x poll: frequent enough for stale-item detection, rare enough to avoid per-tick full-cache churn
	relistDur := 5 * o.PollDur

	eng := engine.New(log.WithName("engine"), o.Pollers)
	inf := informer.New(log.WithName("informer"), eng, o.PollDur, relistDur)
	inf.RegisterHandler(loghandler.NewHandler(log.WithName("event")))

	log.Info("starting", "poll_interval", o.PollDur, "repos", o.Config.Repos, "user", o.User)
	inf.Run(ctx)
	return nil
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
