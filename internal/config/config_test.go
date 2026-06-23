package config

import (
	"os"
	"path/filepath"
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
