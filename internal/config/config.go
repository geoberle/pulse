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

// Config holds the top-level application configuration loaded from
// config.yaml. Fields map 1:1 to YAML keys.
type Config struct {
	// Jira holds connection details for the Jira instance. Required.
	Jira JiraConfig `json:"jira"`

	// Repos is the list of GitHub owner/repo slugs to poll for PRs.
	// Required. Maximum 50 entries. Format: "owner/repo".
	// Example: ["Azure/ARO-HCP", "Azure/ARO-Tools"]
	Repos []string `json:"repos"`

	// JiraProject is the Jira project key used to filter issues
	// (stories and bugs assigned to or created by the user).
	// Required. Maximum 100 characters.
	// Example: "ARO"
	JiraProject string `json:"jira_project"`

	// StaleThreshold is the duration of inactivity after which a Jira
	// issue is marked stale. Must be a valid Go duration string,
	// between 1h and 8760h (365 days). Default: "120h" (5 days).
	// Cannot be explicitly set to empty — omit to use the default.
	// Example: "120h", "48h", "720h"
	StaleThreshold string `json:"stale_threshold"`

	// PollInterval is the duration between upstream poll cycles.
	// Must be a valid Go duration string. Default: "5m".
	// Cannot be explicitly set to empty — omit to use the default.
	// Example: "5m", "30s", "1h"
	PollInterval string `json:"poll_interval"`

	// LLM holds configuration for the LLM provider used for review
	// comment summarization. Optional — when all fields are empty, LLM
	// features are disabled. When Provider is set, Project and Region
	// are required.
	LLM LLMConfig `json:"llm"`
}

// JiraConfig holds connection details for a Jira Cloud instance.
type JiraConfig struct {
	// Host is the base URL of the Jira instance. Required. Must use
	// HTTPS. Maximum 200 characters.
	// Example: "https://redhat.atlassian.net"
	Host string `json:"host"`

	// Email is the Jira account email for API authentication. Required.
	// Maximum 200 characters.
	Email string `json:"email"`

	// Token is the Jira API token (PAT) for authentication. Required.
	// Maximum 500 characters.
	Token string `json:"token"`
}

// LLMConfig holds configuration for the LLM provider used to generate
// review comment summaries. When Provider is set, Project and Region
// are required.
type LLMConfig struct {
	// Provider is the LLM backend identifier. Opaque string — the
	// provider value itself is validated by the LLM client constructor.
	// Maximum 50 characters.
	// Example: "vertex"
	Provider string `json:"provider"`

	// Project is the cloud project ID for the LLM provider.
	// Maximum 100 characters.
	// Example: "my-gcp-project"
	Project string `json:"project"`

	// Region is the cloud region for the LLM endpoint.
	// Maximum 50 characters.
	// Example: "us-east5"
	Region string `json:"region"`
}

// PollDuration parses PollInterval as a time.Duration.
func (c *Config) PollDuration() (time.Duration, error) {
	return time.ParseDuration(c.PollInterval)
}

// StaleDuration parses StaleThreshold as a time.Duration.
func (c *Config) StaleDuration() (time.Duration, error) {
	return time.ParseDuration(c.StaleThreshold)
}

func (c *Config) validateJiraHost() error {
	if len(c.Jira.Host) == 0 {
		return fmt.Errorf("jira.host is required")
	}
	if !strings.HasPrefix(c.Jira.Host, "https://") {
		return fmt.Errorf("jira.host must use https://, got %q", c.Jira.Host)
	}
	if strings.HasSuffix(c.Jira.Host, "/") {
		return fmt.Errorf("jira.host must not have trailing slash, got %q", c.Jira.Host)
	}
	return nil
}

// Validate checks all structural invariants on a loaded Config: required
// fields, value bounds, and format constraints. Call after LoadConfig.
func (c *Config) Validate() error {
	if len(c.Repos) == 0 {
		return fmt.Errorf("repos is required")
	}
	if len(c.Repos) > 50 {
		return fmt.Errorf("repos: max 50 entries, got %d", len(c.Repos))
	}
	for _, repo := range c.Repos {
		if !strings.Contains(repo, "/") {
			return fmt.Errorf("invalid repo format %q, expected owner/repo", repo)
		}
	}
	if len(c.JiraProject) == 0 {
		return fmt.Errorf("jira_project is required")
	}
	if len(c.JiraProject) > 100 {
		return fmt.Errorf("jira_project: max 100 chars, got %d", len(c.JiraProject))
	}
	if !jiraProjectRE.MatchString(c.JiraProject) {
		return fmt.Errorf("jira_project must be uppercase alphanumeric (e.g. ARO), got %q", c.JiraProject)
	}
	if len(c.Jira.Host) > 200 {
		return fmt.Errorf("jira.host: max 200 chars, got %d", len(c.Jira.Host))
	}
	if err := c.validateJiraHost(); err != nil {
		return err
	}
	if len(c.Jira.Email) == 0 {
		return fmt.Errorf("jira.email is required")
	}
	if len(c.Jira.Email) > 200 {
		return fmt.Errorf("jira.email: max 200 chars, got %d", len(c.Jira.Email))
	}
	if len(c.Jira.Token) == 0 {
		return fmt.Errorf("jira.token is required")
	}
	if len(c.Jira.Token) > 500 {
		return fmt.Errorf("jira.token: max 500 chars, got %d", len(c.Jira.Token))
	}
	if len(c.StaleThreshold) == 0 {
		return fmt.Errorf("stale_threshold cannot be empty — omit the field to use the default (120h)")
	}
	staleDur, err := c.StaleDuration()
	if err != nil {
		return fmt.Errorf("invalid stale_threshold: %w", err)
	}
	if staleDur < time.Hour || staleDur > 8760*time.Hour {
		return fmt.Errorf("stale_threshold must be 1h-8760h, got %s", c.StaleThreshold)
	}
	if len(c.PollInterval) == 0 {
		return fmt.Errorf("poll_interval cannot be empty — omit the field to use the default (5m)")
	}
	pollDur, err := c.PollDuration()
	if err != nil {
		return fmt.Errorf("invalid poll_interval: %w", err)
	}
	if pollDur < 30*time.Second {
		return fmt.Errorf("poll_interval must be at least 30s, got %s", c.PollInterval)
	}
	if len(c.LLM.Provider) > 0 {
		if len(c.LLM.Project) == 0 {
			return fmt.Errorf("llm.project is required when llm.provider is set")
		}
		if len(c.LLM.Region) == 0 {
			return fmt.Errorf("llm.region is required when llm.provider is set")
		}
	}
	if len(c.LLM.Provider) > 50 {
		return fmt.Errorf("llm.provider: max 50 chars, got %d", len(c.LLM.Provider))
	}
	if len(c.LLM.Project) > 100 {
		return fmt.Errorf("llm.project: max 100 chars, got %d", len(c.LLM.Project))
	}
	if len(c.LLM.Region) > 50 {
		return fmt.Errorf("llm.region: max 50 chars, got %d", len(c.LLM.Region))
	}
	return nil
}

// LoadConfig reads and parses a config YAML file. Applies defaults for
// StaleThreshold ("120h") and PollInterval ("5m") before unmarshaling.
// These fields cannot be explicitly set to empty — omit them to get
// the default, or set a non-empty value.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	cfg := &Config{
		StaleThreshold: "120h",
		PollInterval:   "5m",
	}
	if err := yaml.UnmarshalStrict(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	return cfg, nil
}

// DefaultConfigPath returns the XDG-compliant default path for the
// application configuration file.
func DefaultConfigPath() (string, error) {
	dir, err := defaultConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
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
