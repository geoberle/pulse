package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds the top-level application configuration loaded from
// config.yaml. Fields map 1:1 to YAML keys.
type Config struct {
	// Jira holds connection details for the Jira instance. Required.
	Jira JiraConfig `yaml:"jira"`

	// Repos is the list of GitHub owner/repo slugs to poll for PRs.
	// Required. Maximum 50 entries.
	// Example: ["Azure/ARO-HCP", "Azure/ARO-Tools"]
	Repos []string `yaml:"repos"`

	// JiraProject is the Jira project key used to filter issues
	// (stories and bugs assigned to or created by the user).
	// Required. Maximum 100 characters.
	// Example: "ARO"
	JiraProject string `yaml:"jira_project"`

	// StaleThresholdDays is the number of days of inactivity after which
	// a Jira issue is marked stale. Must be 1-365. Default: 5.
	// Cannot be explicitly set to zero — omit to use the default.
	StaleThresholdDays int `yaml:"stale_threshold_days"`

	// PollInterval is the duration between upstream poll cycles.
	// Must be a valid Go duration string. Default: "5m".
	// Cannot be explicitly set to empty — omit to use the default.
	// Example: "5m", "30s", "1h"
	PollInterval string `yaml:"poll_interval"`

	// LLM holds configuration for the LLM provider used for review
	// comment summarization. Optional — when omitted, LLM features
	// are disabled. Validated lazily when the LLM client is constructed,
	// not at config load time.
	LLM LLMConfig `yaml:"llm"`
}

// JiraConfig holds connection details for a Jira Cloud instance.
type JiraConfig struct {
	// Host is the base URL of the Jira instance. Required. Must use
	// HTTPS. Maximum 200 characters.
	// Example: "https://redhat.atlassian.net"
	Host string `yaml:"host"`

	// Email is the Jira account email for API authentication.
	// Optional in config — may be sourced from environment or keychain.
	// Maximum 200 characters.
	Email string `yaml:"email"`

	// Token is the Jira API token (PAT) for authentication.
	// Optional in config — may be sourced from environment or keychain.
	// Maximum 500 characters.
	Token string `yaml:"token"`
}

// LLMConfig holds configuration for the LLM provider used to generate
// review comment summaries. Validated lazily when the LLM client is
// constructed, not at config load time.
type LLMConfig struct {
	// Provider is the LLM backend identifier. Maximum 50 characters.
	// Example: "vertex"
	Provider string `yaml:"provider"`

	// Project is the cloud project ID for the LLM provider.
	// Maximum 100 characters.
	// Example: "my-gcp-project"
	Project string `yaml:"project"`

	// Region is the cloud region for the LLM endpoint.
	// Maximum 50 characters.
	// Example: "us-east5"
	Region string `yaml:"region"`
}

// PollDuration parses PollInterval as a time.Duration.
func (c *Config) PollDuration() (time.Duration, error) {
	return time.ParseDuration(c.PollInterval)
}

// ValidateJiraHost checks that the Jira host is non-empty and uses HTTPS.
func (c *Config) ValidateJiraHost() error {
	if len(c.Jira.Host) == 0 {
		return fmt.Errorf("jira.host is required")
	}
	if !strings.HasPrefix(c.Jira.Host, "https://") {
		return fmt.Errorf("jira.host must use https://, got %q", c.Jira.Host)
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
	if len(c.JiraProject) == 0 {
		return fmt.Errorf("jira_project is required")
	}
	if len(c.JiraProject) > 100 {
		return fmt.Errorf("jira_project: max 100 chars, got %d", len(c.JiraProject))
	}
	if err := c.ValidateJiraHost(); err != nil {
		return err
	}
	if len(c.Jira.Host) > 200 {
		return fmt.Errorf("jira.host: max 200 chars, got %d", len(c.Jira.Host))
	}
	if len(c.Jira.Email) > 200 {
		return fmt.Errorf("jira.email: max 200 chars, got %d", len(c.Jira.Email))
	}
	if len(c.Jira.Token) > 500 {
		return fmt.Errorf("jira.token: max 500 chars, got %d", len(c.Jira.Token))
	}
	if c.StaleThresholdDays < 1 || c.StaleThresholdDays > 365 {
		return fmt.Errorf("stale_threshold_days must be 1-365, got %d", c.StaleThresholdDays)
	}
	if _, err := c.PollDuration(); err != nil {
		return fmt.Errorf("invalid poll_interval: %w", err)
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
// StaleThresholdDays (5) and PollInterval ("5m") before unmarshaling.
// These fields cannot be explicitly set to zero — omit them to get
// the default, or set a non-zero value.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	cfg := &Config{
		StaleThresholdDays: 5,
		PollInterval:       "5m",
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
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
	if xdg := os.Getenv("XDG_CONFIG_HOME"); len(xdg) > 0 {
		return filepath.Join(xdg, "pulse"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determine home directory: %w", err)
	}
	return filepath.Join(home, ".config", "pulse"), nil
}
