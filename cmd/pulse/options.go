package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	jira "github.com/ctreminiom/go-atlassian/v2/jira/v2"
	"github.com/go-logr/logr"
	gogithub "github.com/google/go-github/v72/github"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/cache"

	"github.com/geoberle/pulse/internal/config"
	"github.com/geoberle/pulse/internal/engine"
	loghandler "github.com/geoberle/pulse/internal/handler/log"
	"github.com/geoberle/pulse/internal/informer"
	"github.com/geoberle/pulse/internal/poller"
	ghpoller "github.com/geoberle/pulse/internal/poller/github"
	jirapoller "github.com/geoberle/pulse/internal/poller/jira"
	jsonstore "github.com/geoberle/pulse/internal/store/json"
	"github.com/geoberle/pulse/internal/workitem"
)

// RawOptions holds unvalidated CLI flag values. Populated by cobra flag
// binding before any validation occurs.
type RawOptions struct {
	ConfigFile  string
	PromptsFile string
	StateFile   string
}

type validatedOptions struct {
	Config    *config.Config
	Prompts   *config.Prompts
	StateFile string
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
	Engine  *engine.Engine
	Store   *jsonstore.Store
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
	statePath, err := config.DefaultStatePath()
	if err != nil {
		return nil, err
	}
	return &RawOptions{
		ConfigFile:  cfgPath,
		PromptsFile: promptsPath,
		StateFile:   statePath,
	}, nil
}

// BindFlags registers CLI flags on the cobra command.
func (o *RawOptions) BindFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&o.ConfigFile, "config-file", o.ConfigFile, "Path to config file")
	cmd.Flags().StringVar(&o.PromptsFile, "prompts-file", o.PromptsFile, "Path to prompts file")
	cmd.Flags().StringVar(&o.StateFile, "state-file", o.StateFile, "Path to state file")
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
			Config:    cfg,
			Prompts:   prompts,
			StateFile: o.StateFile,
		},
	}, nil
}

// Complete constructs external clients and finalizes options for execution.
func (o *ValidatedOptions) Complete(ctx context.Context) (*Options, error) {
	pollDur, err := o.Config.PollDuration()
	if err != nil {
		return nil, fmt.Errorf("poll interval: %w", err)
	}
	staleDur, err := o.Config.StaleDuration()
	if err != nil {
		return nil, fmt.Errorf("stale threshold: %w", err)
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

	jiraClient, err := jira.New(nil, o.Config.Jira.Host)
	if err != nil {
		return nil, fmt.Errorf("jira client: %w", err)
	}
	jiraClient.Auth.SetBasicAuth(o.Config.Jira.Email, o.Config.Jira.Token)
	jiraPoll := jirapoller.NewPoller(jiraClient.Issue.Search, o.Config.JiraProject, staleDur)

	log, _ := logr.FromContext(ctx)
	eng := engine.New(log.WithName("engine"), []poller.Poller{ghPoll, jiraPoll})

	store, err := jsonstore.New(o.StateFile, log.WithName("store"))
	if err != nil {
		return nil, fmt.Errorf("state store: %w", err)
	}

	return &Options{
		completedOptions: completedOptions{
			Config:  o.Config,
			Prompts: o.Prompts,
			Engine:  eng,
			Store:   store,
			PollDur: pollDur,
		},
	}, nil
}

// cachedSource wraps a live Source, returning persisted state on the
// first List call for instant cache population. Subsequent calls
// delegate to the live source; the reflector diffs against the seeded
// cache and only emits events for real changes.
type cachedSource struct {
	mu     sync.Mutex
	store  *jsonstore.Store
	live   informer.Source
	log    logr.Logger
	seeded bool
	liveC  chan struct{}
}

// List is called exclusively by the reflector (single-threaded).
func (s *cachedSource) List(ctx context.Context) ([]*workitem.WorkItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.seeded {
		s.seeded = true
		items, err := s.store.Load()
		if err != nil {
			s.log.Error(err, "loading persisted state, falling through to live source")
		} else if len(items) > 0 {
			return items, nil
		}
	}
	result, err := s.live.List(ctx)
	if err == nil {
		select {
		case <-s.liveC:
		default:
			close(s.liveC)
		}
	}
	return result, err
}

func (s *cachedSource) LiveReady() <-chan struct{} { return s.liveC }

// Run executes the main application loop.
func (o *Options) Run(ctx context.Context) error {
	log, _ := logr.FromContext(ctx)

	src := &cachedSource{
		store: o.Store,
		live:  o.Engine,
		log:   log.WithName("cache"),
		liveC: make(chan struct{}),
	}
	inf := informer.New(src, o.PollDur)
	if _, err := inf.AddEventHandler(loghandler.NewHandler(log.WithName("event"))); err != nil {
		return fmt.Errorf("add event handler: %w", err)
	}

	lister := informer.NewLister(inf.GetIndexer())

	stopCh := ctx.Done()
	go inf.Run(stopCh)

	log.Info("starting", "poll_interval", o.PollDur)
	if !cache.WaitForCacheSync(stopCh, inf.HasSynced) {
		return fmt.Errorf("informer sync cancelled")
	}
	log.Info("informer synced")

	go o.persistPeriodically(ctx, log, lister, src.LiveReady())

	<-ctx.Done()
	return nil
}

func (o *Options) persistPeriodically(ctx context.Context, log logr.Logger, lister informer.Lister, ready <-chan struct{}) {
	select {
	case <-ready:
	case <-ctx.Done():
		return
	}
	o.persist(log, lister)
	ticker := time.NewTicker(o.PollDur)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			o.persist(log, lister)
		}
	}
}

// persist snapshots the informer cache to disk. Lister returns shared
// pointers (K8s convention); BuildTree DeepCopies before mutating.
func (o *Options) persist(log logr.Logger, lister informer.Lister) {
	items, err := lister.List()
	if err != nil {
		log.Error(err, "failed to list items for persistence")
		return
	}
	tree := workitem.BuildTree(items)
	if err := o.Store.Save(tree); err != nil {
		log.Error(err, "failed to persist state")
	}
}
