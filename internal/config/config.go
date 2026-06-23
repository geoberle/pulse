package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Jira               JiraConfig `yaml:"jira"`
	Repos              []string   `yaml:"repos"`
	JiraProject        string     `yaml:"jira_project"`
	StaleThresholdDays int        `yaml:"stale_threshold_days"`
	PollInterval       string     `yaml:"poll_interval"`
	LLM                LLMConfig  `yaml:"llm"`
}

type JiraConfig struct {
	Host  string `yaml:"host"`
	Email string `yaml:"email"`
	Token string `yaml:"token"`
}

type LLMConfig struct {
	Provider string `yaml:"provider"`
	Project  string `yaml:"project"`
	Region   string `yaml:"region"`
}

func (c *Config) PollDuration() (time.Duration, error) {
	return time.ParseDuration(c.PollInterval)
}

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
