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
	// Jira holds connection details for the Jira instance.
	Jira JiraConfig `yaml:"jira"`

	// Repos is the list of GitHub owner/repo slugs to poll for PRs,
	// e.g. ["Azure/ARO-HCP", "Azure/ARO-Tools"].
	Repos []string `yaml:"repos"`

	// JiraProject is the Jira project key used to filter issues,
	// e.g. "ARO".
	JiraProject string `yaml:"jira_project"`

	// StaleThresholdDays is the number of days of inactivity after which
	// a Jira issue is marked stale. Defaults to 5.
	StaleThresholdDays int `yaml:"stale_threshold_days"`

	// PollInterval is the duration string (e.g. "5m") between upstream
	// poll cycles. Parsed via time.ParseDuration.
	PollInterval string `yaml:"poll_interval"`

	// LLM holds configuration for the LLM provider used for review
	// comment summarization.
	LLM LLMConfig `yaml:"llm"`
}

// JiraConfig holds connection details for a Jira Cloud instance.
type JiraConfig struct {
	// Host is the base URL of the Jira instance, e.g.
	// "https://redhat.atlassian.net". Must use HTTPS.
	Host string `yaml:"host"`

	// Email is the Jira account email used for API authentication.
	// May be empty if sourced from environment or keychain.
	Email string `yaml:"email"`

	// Token is the Jira API token (PAT) for authentication.
	// May be empty if sourced from environment or keychain.
	Token string `yaml:"token"`
}

// LLMConfig holds configuration for the LLM provider used to generate
// review comment summaries.
type LLMConfig struct {
	// Provider is the LLM backend identifier, e.g. "vertex".
	Provider string `yaml:"provider"`

	// Project is the cloud project ID for the LLM provider.
	Project string `yaml:"project"`

	// Region is the cloud region for the LLM endpoint, e.g. "us-east5".
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

// LoadConfig reads and parses a config YAML file. Applies defaults for
// StaleThresholdDays (5) and PollInterval ("5m") before unmarshaling.
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
