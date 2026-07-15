package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"sigs.k8s.io/yaml"
)

var jiraProjectRE = regexp.MustCompile(`^[A-Z][A-Z0-9_]+$`)

type Config struct {
	Repos          []RepoConfig `json:"repos"`
	Jira           JiraConfig   `json:"jira"`
	LLM            LLMConfig    `json:"llm"`
	PollIntervals  PollConfig   `json:"poll_intervals"`
	StaleThreshold string       `json:"stale_threshold"`
}

type RepoConfig struct {
	Owner string `json:"owner"`
	Name  string `json:"name"`
	Path  string `json:"path"`
}

type JiraConfig struct {
	Host             string `json:"host"`
	Project          string `json:"project"`
	Component        string `json:"component"`
	DefaultIssueType string `json:"default_issue_type"`
}

type LLMConfig struct {
	Provider string `json:"provider"`
	Project  string `json:"project"`
	Region   string `json:"region"`
}

type PollConfig struct {
	Git    string `json:"git"`
	GitHub string `json:"github"`
	Jira   string `json:"jira"`
}

func (c *PollConfig) GitDuration() (time.Duration, error) {
	return time.ParseDuration(c.Git)
}

func (c *PollConfig) GitHubDuration() (time.Duration, error) {
	return time.ParseDuration(c.GitHub)
}

func (c *PollConfig) JiraDuration() (time.Duration, error) {
	return time.ParseDuration(c.Jira)
}

func (c *Config) StaleDuration() (time.Duration, error) {
	return time.ParseDuration(c.StaleThreshold)
}

func (c *Config) Validate() error {
	if len(c.Repos) == 0 {
		return fmt.Errorf("repos is required")
	}
	for i, r := range c.Repos {
		if len(r.Owner) == 0 {
			return fmt.Errorf("repos[%d].owner is required", i)
		}
		if len(r.Name) == 0 {
			return fmt.Errorf("repos[%d].name is required", i)
		}
		if len(r.Path) == 0 {
			return fmt.Errorf("repos[%d].path is required", i)
		}
		expanded := expandTilde(r.Path)
		info, err := os.Stat(expanded)
		if err != nil {
			return fmt.Errorf("repos[%d].path %q: %w", i, r.Path, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("repos[%d].path %q is not a directory", i, r.Path)
		}
	}

	if err := c.validateJira(); err != nil {
		return err
	}
	if err := c.validatePollIntervals(); err != nil {
		return err
	}
	if err := c.validateStaleThreshold(); err != nil {
		return err
	}
	if err := c.validateLLM(); err != nil {
		return err
	}
	return nil
}

func (c *Config) validateJira() error {
	if len(c.Jira.Host) == 0 {
		return fmt.Errorf("jira.host is required")
	}
	if !strings.HasPrefix(c.Jira.Host, "https://") {
		return fmt.Errorf("jira.host must use https://, got %q", c.Jira.Host)
	}
	if strings.HasSuffix(c.Jira.Host, "/") {
		return fmt.Errorf("jira.host must not have trailing slash, got %q", c.Jira.Host)
	}
	if len(c.Jira.Project) == 0 {
		return fmt.Errorf("jira.project is required")
	}
	if !jiraProjectRE.MatchString(c.Jira.Project) {
		return fmt.Errorf("jira.project must match [A-Z][A-Z0-9_]+, got %q", c.Jira.Project)
	}
	if len(c.Jira.Component) == 0 {
		return fmt.Errorf("jira.component is required")
	}
	if len(c.Jira.DefaultIssueType) == 0 {
		return fmt.Errorf("jira.default_issue_type is required")
	}
	return nil
}

func (c *Config) validatePollIntervals() error {
	gitDur, err := c.PollIntervals.GitDuration()
	if err != nil {
		return fmt.Errorf("poll_intervals.git: %w", err)
	}
	if gitDur < 10*time.Second {
		return fmt.Errorf("poll_intervals.git must be at least 10s, got %s", c.PollIntervals.Git)
	}

	ghDur, err := c.PollIntervals.GitHubDuration()
	if err != nil {
		return fmt.Errorf("poll_intervals.github: %w", err)
	}
	if ghDur < time.Minute {
		return fmt.Errorf("poll_intervals.github must be at least 1m, got %s", c.PollIntervals.GitHub)
	}

	jiraDur, err := c.PollIntervals.JiraDuration()
	if err != nil {
		return fmt.Errorf("poll_intervals.jira: %w", err)
	}
	if jiraDur < time.Minute {
		return fmt.Errorf("poll_intervals.jira must be at least 1m, got %s", c.PollIntervals.Jira)
	}

	return nil
}

func (c *Config) validateStaleThreshold() error {
	if len(c.StaleThreshold) == 0 {
		return fmt.Errorf("stale_threshold cannot be empty — omit to use default (120h)")
	}
	dur, err := c.StaleDuration()
	if err != nil {
		return fmt.Errorf("stale_threshold: %w", err)
	}
	if dur < time.Hour || dur > 8760*time.Hour {
		return fmt.Errorf("stale_threshold must be 1h-8760h, got %s", c.StaleThreshold)
	}
	return nil
}

func (c *Config) validateLLM() error {
	if len(c.LLM.Provider) == 0 {
		return nil
	}
	if len(c.LLM.Project) == 0 {
		return fmt.Errorf("llm.project is required when llm.provider is set")
	}
	if len(c.LLM.Region) == 0 {
		return fmt.Errorf("llm.region is required when llm.provider is set")
	}
	return nil
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	cfg := &Config{
		PollIntervals: PollConfig{
			Git:    "30s",
			GitHub: "5m",
			Jira:   "5m",
		},
		StaleThreshold: "120h",
	}
	if err := yaml.UnmarshalStrict(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	return cfg, nil
}

func DefaultConfigPath() (string, error) {
	dir, err := defaultConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

func DefaultStateDir() (string, error) {
	if xdg := os.Getenv("XDG_STATE_HOME"); len(xdg) != 0 {
		return filepath.Join(xdg, "pulse"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determine home directory: %w", err)
	}
	return filepath.Join(home, ".local", "state", "pulse"), nil
}

func defaultConfigDir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); len(xdg) != 0 {
		return filepath.Join(xdg, "pulse"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determine home directory: %w", err)
	}
	return filepath.Join(home, ".config", "pulse"), nil
}

func expandTilde(path string) string {
	if !strings.HasPrefix(path, "~/") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[2:])
}
