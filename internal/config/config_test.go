package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	t.Parallel()
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
	if cfg.StaleThreshold != "120h" {
		t.Errorf("expected stale_threshold 120h, got %s", cfg.StaleThreshold)
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
	t.Parallel()
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
	if cfg.StaleThreshold != "120h" {
		t.Errorf("expected default stale_threshold 120h, got %s", cfg.StaleThreshold)
	}
}

func TestLoadConfig_Errors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "missing file",
			content: "",
		},
		{
			name:    "invalid YAML",
			content: ":\ninvalid: [yaml\n",
		},
		{
			name:    "unknown key",
			content: "jira_project: TEST\nunknown_field: oops\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var path string
			if len(tt.content) == 0 {
				path = "/nonexistent/config.yaml"
			} else {
				tmp := t.TempDir()
				path = filepath.Join(tmp, "config.yaml")
				if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
					t.Fatal(err)
				}
			}
			_, err := LoadConfig(path)
			if err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestDefaultConfigPath_XDG(t *testing.T) {
	// t.Setenv is incompatible with t.Parallel
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg-test")
	path, err := DefaultConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	if path != "/tmp/xdg-test/pulse/config.yaml" {
		t.Errorf("expected /tmp/xdg-test/pulse/config.yaml, got %s", path)
	}
}

func TestValidate(t *testing.T) {
	t.Parallel()
	validCfg := func() Config {
		return Config{
			Jira:           JiraConfig{Host: "https://example.atlassian.net"},
			Repos:          []string{"org/repo"},
			JiraProject:    "PROJ",
			StaleThreshold: "120h",
			PollInterval:   "5m",
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
			name: "stale_threshold empty",
			cfg: func() Config {
				c := validCfg()
				c.StaleThreshold = ""
				return c
			}(),
			wantErr: true,
		},
		{
			name: "stale_threshold invalid",
			cfg: func() Config {
				c := validCfg()
				c.StaleThreshold = "notaduration"
				return c
			}(),
			wantErr: true,
		},
		{
			name: "stale_threshold too low",
			cfg: func() Config {
				c := validCfg()
				c.StaleThreshold = "30m"
				return c
			}(),
			wantErr: true,
		},
		{
			name: "stale_threshold too high",
			cfg: func() Config {
				c := validCfg()
				c.StaleThreshold = "8761h"
				return c
			}(),
			wantErr: true,
		},
		{
			name: "stale_threshold boundary 1h",
			cfg: func() Config {
				c := validCfg()
				c.StaleThreshold = "1h"
				return c
			}(),
		},
		{
			name: "stale_threshold boundary 8760h",
			cfg: func() Config {
				c := validCfg()
				c.StaleThreshold = "8760h"
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
		{
			name: "poll_interval too low",
			cfg: func() Config {
				c := validCfg()
				c.PollInterval = "1s"
				return c
			}(),
			wantErr: true,
		},
		{
			name: "poll_interval boundary 30s",
			cfg: func() Config {
				c := validCfg()
				c.PollInterval = "30s"
				return c
			}(),
		},
		{
			name: "invalid repo format",
			cfg: func() Config {
				c := validCfg()
				c.Repos = []string{"noslash"}
				return c
			}(),
			wantErr: true,
		},
		{
			name: "llm provider set without project",
			cfg: func() Config {
				c := validCfg()
				c.LLM = LLMConfig{Provider: "vertex", Region: "us-east5"}
				return c
			}(),
			wantErr: true,
		},
		{
			name: "llm provider set without region",
			cfg: func() Config {
				c := validCfg()
				c.LLM = LLMConfig{Provider: "vertex", Project: "my-project"}
				return c
			}(),
			wantErr: true,
		},
		{
			name: "llm fully configured",
			cfg: func() Config {
				c := validCfg()
				c.LLM = LLMConfig{Provider: "vertex", Project: "proj", Region: "us-east5"}
				return c
			}(),
		},
		{
			name: "llm all empty (disabled)",
			cfg: func() Config {
				c := validCfg()
				c.LLM = LLMConfig{}
				return c
			}(),
		},
		{
			name: "jira host too long",
			cfg: func() Config {
				c := validCfg()
				c.Jira.Host = "https://" + strings.Repeat("a", 193)
				return c
			}(),
			wantErr: true,
		},
		{
			name: "jira email too long",
			cfg: func() Config {
				c := validCfg()
				c.Jira.Email = strings.Repeat("a", 201)
				return c
			}(),
			wantErr: true,
		},
		{
			name: "jira token too long",
			cfg: func() Config {
				c := validCfg()
				c.Jira.Token = strings.Repeat("a", 501)
				return c
			}(),
			wantErr: true,
		},
		{
			name: "llm provider too long",
			cfg: func() Config {
				c := validCfg()
				c.LLM.Provider = strings.Repeat("a", 51)
				return c
			}(),
			wantErr: true,
		},
		{
			name: "llm project too long",
			cfg: func() Config {
				c := validCfg()
				c.LLM.Project = strings.Repeat("a", 101)
				return c
			}(),
			wantErr: true,
		},
		{
			name: "llm region too long",
			cfg: func() Config {
				c := validCfg()
				c.LLM.Region = strings.Repeat("a", 51)
				return c
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateJiraHost(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		host    string
		wantErr bool
	}{
		{
			name: "valid https host",
			host: "https://redhat.atlassian.net",
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
			t.Parallel()
			cfg := &Config{Jira: JiraConfig{Host: tt.host}}
			err := cfg.ValidateJiraHost()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateJiraHost() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
