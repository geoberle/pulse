package config

import (
	"os"
	"path/filepath"
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
	if cfg.Repos[0].Owner != "Azure" {
		t.Errorf("expected first repo owner Azure, got %s", cfg.Repos[0].Owner)
	}
	if cfg.Repos[0].Name != "ARO-HCP" {
		t.Errorf("expected first repo name ARO-HCP, got %s", cfg.Repos[0].Name)
	}
	if cfg.Jira.Project != "ARO" {
		t.Errorf("expected jira.project ARO, got %s", cfg.Jira.Project)
	}
	if cfg.StaleThreshold != "120h" {
		t.Errorf("expected stale_threshold 120h, got %s", cfg.StaleThreshold)
	}
	if cfg.PollIntervals.Git != "30s" {
		t.Errorf("expected poll_intervals.git 30s, got %s", cfg.PollIntervals.Git)
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.yaml")
	if err := os.WriteFile(path, []byte("jira:\n  host: https://x.atlassian.net\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PollIntervals.Git != "30s" {
		t.Errorf("expected default git poll 30s, got %s", cfg.PollIntervals.Git)
	}
	if cfg.PollIntervals.GitHub != "5m" {
		t.Errorf("expected default github poll 5m, got %s", cfg.PollIntervals.GitHub)
	}
	if cfg.PollIntervals.Jira != "5m" {
		t.Errorf("expected default jira poll 5m, got %s", cfg.PollIntervals.Jira)
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
		{name: "missing file", content: ""},
		{name: "invalid YAML", content: ":\ninvalid: [yaml\n"},
		{name: "unknown key", content: "jira:\n  host: https://x.atlassian.net\nunknown_field: oops\n"},
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

func TestValidate(t *testing.T) {
	t.Parallel()

	validDir := t.TempDir()

	validCfg := func() Config {
		return Config{
			Repos: []RepoConfig{
				{Owner: "Azure", Name: "ARO-HCP", Path: validDir},
			},
			Jira: JiraConfig{
				Host:             "https://redhat.atlassian.net",
				Project:          "ARO",
				Component:        "aro-hcp-1p",
				DefaultIssueType: "Story",
			},
			PollIntervals: PollConfig{
				Git:    "30s",
				GitHub: "5m",
				Jira:   "5m",
			},
			StaleThreshold: "120h",
		}
	}

	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{name: "valid config", cfg: validCfg()},

		// repos
		{name: "no repos", cfg: func() Config { c := validCfg(); c.Repos = nil; return c }(), wantErr: true},
		{name: "empty repos", cfg: func() Config { c := validCfg(); c.Repos = []RepoConfig{}; return c }(), wantErr: true},
		{name: "missing owner", cfg: func() Config {
			c := validCfg()
			c.Repos[0].Owner = ""
			return c
		}(), wantErr: true},
		{name: "missing name", cfg: func() Config {
			c := validCfg()
			c.Repos[0].Name = ""
			return c
		}(), wantErr: true},
		{name: "missing path", cfg: func() Config {
			c := validCfg()
			c.Repos[0].Path = ""
			return c
		}(), wantErr: true},
		{name: "nonexistent path", cfg: func() Config {
			c := validCfg()
			c.Repos[0].Path = "/nonexistent/path"
			return c
		}(), wantErr: true},

		// jira
		{name: "missing jira host", cfg: func() Config { c := validCfg(); c.Jira.Host = ""; return c }(), wantErr: true},
		{name: "http jira host", cfg: func() Config { c := validCfg(); c.Jira.Host = "http://x.atlassian.net"; return c }(), wantErr: true},
		{name: "trailing slash jira host", cfg: func() Config { c := validCfg(); c.Jira.Host = "https://x.atlassian.net/"; return c }(), wantErr: true},
		{name: "missing jira project", cfg: func() Config { c := validCfg(); c.Jira.Project = ""; return c }(), wantErr: true},
		{name: "lowercase jira project", cfg: func() Config { c := validCfg(); c.Jira.Project = "aro"; return c }(), wantErr: true},
		{name: "missing jira component", cfg: func() Config { c := validCfg(); c.Jira.Component = ""; return c }(), wantErr: true},
		{name: "missing jira default_issue_type", cfg: func() Config { c := validCfg(); c.Jira.DefaultIssueType = ""; return c }(), wantErr: true},

		// poll intervals
		{name: "invalid git duration", cfg: func() Config { c := validCfg(); c.PollIntervals.Git = "bad"; return c }(), wantErr: true},
		{name: "git too low", cfg: func() Config { c := validCfg(); c.PollIntervals.Git = "5s"; return c }(), wantErr: true},
		{name: "git boundary 10s", cfg: func() Config { c := validCfg(); c.PollIntervals.Git = "10s"; return c }()},
		{name: "invalid github duration", cfg: func() Config { c := validCfg(); c.PollIntervals.GitHub = "bad"; return c }(), wantErr: true},
		{name: "github too low", cfg: func() Config { c := validCfg(); c.PollIntervals.GitHub = "30s"; return c }(), wantErr: true},
		{name: "github boundary 1m", cfg: func() Config { c := validCfg(); c.PollIntervals.GitHub = "1m"; return c }()},
		{name: "invalid jira duration", cfg: func() Config { c := validCfg(); c.PollIntervals.Jira = "bad"; return c }(), wantErr: true},
		{name: "jira too low", cfg: func() Config { c := validCfg(); c.PollIntervals.Jira = "30s"; return c }(), wantErr: true},
		{name: "jira boundary 1m", cfg: func() Config { c := validCfg(); c.PollIntervals.Jira = "1m"; return c }()},

		// stale threshold
		{name: "empty stale_threshold", cfg: func() Config { c := validCfg(); c.StaleThreshold = ""; return c }(), wantErr: true},
		{name: "invalid stale_threshold", cfg: func() Config { c := validCfg(); c.StaleThreshold = "bad"; return c }(), wantErr: true},
		{name: "stale_threshold too low", cfg: func() Config { c := validCfg(); c.StaleThreshold = "30m"; return c }(), wantErr: true},
		{name: "stale_threshold too high", cfg: func() Config { c := validCfg(); c.StaleThreshold = "8761h"; return c }(), wantErr: true},
		{name: "stale_threshold boundary 1h", cfg: func() Config { c := validCfg(); c.StaleThreshold = "1h"; return c }()},
		{name: "stale_threshold boundary 8760h", cfg: func() Config { c := validCfg(); c.StaleThreshold = "8760h"; return c }()},

		// llm
		{name: "llm all empty (disabled)", cfg: func() Config { c := validCfg(); c.LLM = LLMConfig{}; return c }()},
		{name: "llm fully configured", cfg: func() Config {
			c := validCfg()
			c.LLM = LLMConfig{Provider: "vertex", Project: "proj", Region: "us-east5"}
			return c
		}()},
		{name: "llm provider without project", cfg: func() Config {
			c := validCfg()
			c.LLM = LLMConfig{Provider: "vertex", Region: "us-east5"}
			return c
		}(), wantErr: true},
		{name: "llm provider without region", cfg: func() Config {
			c := validCfg()
			c.LLM = LLMConfig{Provider: "vertex", Project: "proj"}
			return c
		}(), wantErr: true},
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

func TestDefaultConfigPath_HomeDir(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	path, err := DefaultConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	expected := filepath.Join(home, ".config", "pulse", "config.yaml")
	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}

func TestDefaultStateDir_XDG(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/tmp/xdg-state")
	path, err := DefaultStateDir()
	if err != nil {
		t.Fatal(err)
	}
	if path != "/tmp/xdg-state/pulse" {
		t.Errorf("expected /tmp/xdg-state/pulse, got %s", path)
	}
}

func TestDefaultStateDir_HomeDir(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "")
	path, err := DefaultStateDir()
	if err != nil {
		t.Fatal(err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	expected := filepath.Join(home, ".local", "state", "pulse")
	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}

func TestExpandTilde(t *testing.T) {
	t.Parallel()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "tilde prefix", in: "~/dev/repo", want: filepath.Join(home, "dev", "repo")},
		{name: "no tilde", in: "/abs/path", want: "/abs/path"},
		{name: "tilde only", in: "~", want: "~"},
		{name: "relative", in: "relative/path", want: "relative/path"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := expandTilde(tt.in)
			if got != tt.want {
				t.Errorf("expandTilde(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
