package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join("..", "..", "config", "default_config.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	if len(cfg.Repos) != 2 {
		t.Errorf("expected 2 repos, got %d", len(cfg.Repos))
	}
	if cfg.Repos[0] != "Azure/ARO-HCP" {
		t.Errorf("expected first repo Azure/ARO-HCP, got %s", cfg.Repos[0])
	}
	if cfg.JiraProject != "ARO" {
		t.Errorf("expected jira_project ARO, got %s", cfg.JiraProject)
	}
	if cfg.StaleThresholdDays != 5 {
		t.Errorf("expected stale_threshold_days 5, got %d", cfg.StaleThresholdDays)
	}

	d, err := cfg.PollDuration()
	if err != nil {
		t.Fatal(err)
	}
	if d.Minutes() != 5 {
		t.Errorf("expected 5m poll interval, got %v", d)
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.yaml")
	if err := os.WriteFile(path, []byte("jira_project: TEST\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.JiraProject != "TEST" {
		t.Errorf("expected jira_project TEST, got %s", cfg.JiraProject)
	}
	if cfg.StaleThresholdDays != 5 {
		t.Errorf("expected default stale_threshold_days 5, got %d", cfg.StaleThresholdDays)
	}
}

func TestLoadConfig_MissingFile(t *testing.T) {
	_, err := LoadConfig("/nonexistent/config.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestDefaultConfigPath_XDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg-test")
	path, err := DefaultConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	if path != "/tmp/xdg-test/pulse/config.yaml" {
		t.Errorf("expected /tmp/xdg-test/pulse/config.yaml, got %s", path)
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.yaml")
	if err := os.WriteFile(path, []byte(":\ninvalid: [yaml\n"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadConfig(path)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestValidate(t *testing.T) {
	validCfg := func() Config {
		return Config{
			Jira:               JiraConfig{Host: "https://example.atlassian.net"},
			Repos:              []string{"org/repo"},
			JiraProject:        "PROJ",
			StaleThresholdDays: 5,
			PollInterval:       "5m",
		}
	}

	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg:  validCfg(),
		},
		{
			name: "no repos",
			cfg: func() Config {
				c := validCfg()
				c.Repos = nil
				return c
			}(),
			wantErr: true,
		},
		{
			name: "too many repos",
			cfg: func() Config {
				c := validCfg()
				c.Repos = make([]string, 51)
				return c
			}(),
			wantErr: true,
		},
		{
			name: "missing jira_project",
			cfg: func() Config {
				c := validCfg()
				c.JiraProject = ""
				return c
			}(),
			wantErr: true,
		},
		{
			name: "jira_project too long",
			cfg: func() Config {
				c := validCfg()
				c.JiraProject = strings.Repeat("A", 101)
				return c
			}(),
			wantErr: true,
		},
		{
			name: "stale_threshold_days zero",
			cfg: func() Config {
				c := validCfg()
				c.StaleThresholdDays = 0
				return c
			}(),
			wantErr: true,
		},
		{
			name: "stale_threshold_days negative",
			cfg: func() Config {
				c := validCfg()
				c.StaleThresholdDays = -1
				return c
			}(),
			wantErr: true,
		},
		{
			name: "stale_threshold_days too high",
			cfg: func() Config {
				c := validCfg()
				c.StaleThresholdDays = 366
				return c
			}(),
			wantErr: true,
		},
		{
			name: "stale_threshold_days boundary 1",
			cfg: func() Config {
				c := validCfg()
				c.StaleThresholdDays = 1
				return c
			}(),
		},
		{
			name: "stale_threshold_days boundary 365",
			cfg: func() Config {
				c := validCfg()
				c.StaleThresholdDays = 365
				return c
			}(),
		},
		{
			name: "invalid poll_interval",
			cfg: func() Config {
				c := validCfg()
				c.PollInterval = "notaduration"
				return c
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateJiraHost(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		wantErr bool
	}{
		{
			name:    "valid https host",
			host:    "https://redhat.atlassian.net",
			wantErr: false,
		},
		{
			name:    "empty host",
			host:    "",
			wantErr: true,
		},
		{
			name:    "http host",
			host:    "http://redhat.atlassian.net",
			wantErr: true,
		},
		{
			name:    "no scheme",
			host:    "redhat.atlassian.net",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Jira: JiraConfig{Host: tt.host}}
			err := cfg.ValidateJiraHost()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateJiraHost() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
