package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestValidate_MissingConfigFile(t *testing.T) {
	opts := &RawOptions{
		ConfigFile:  "/nonexistent/config.yaml",
		PromptsFile: "/nonexistent/prompts.yaml",
	}
	_, err := opts.Validate(context.Background())
	if err == nil {
		t.Error("expected error for missing config file")
	}
}

func TestValidate_NoRepos(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "config.yaml")
	promptsPath := filepath.Join(tmp, "prompts.yaml")

	if err := os.WriteFile(cfgPath, []byte("jira_project: ARO\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(promptsPath, []byte("review_comment:\n  prompt: test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	opts := &RawOptions{ConfigFile: cfgPath, PromptsFile: promptsPath}
	_, err := opts.Validate(context.Background())
	if err == nil {
		t.Error("expected error for no repos")
	}
}

func TestValidate_MissingJiraProject(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "config.yaml")
	promptsPath := filepath.Join(tmp, "prompts.yaml")

	if err := os.WriteFile(cfgPath, []byte("repos:\n  - Azure/ARO-HCP\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(promptsPath, []byte("review_comment:\n  prompt: test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	opts := &RawOptions{ConfigFile: cfgPath, PromptsFile: promptsPath}
	_, err := opts.Validate(context.Background())
	if err == nil {
		t.Error("expected error for missing jira_project")
	}
}

func TestValidate_InvalidPollInterval(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "config.yaml")
	promptsPath := filepath.Join(tmp, "prompts.yaml")

	if err := os.WriteFile(cfgPath, []byte("repos:\n  - Azure/ARO-HCP\njira_project: ARO\npoll_interval: notaduration\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(promptsPath, []byte("review_comment:\n  prompt: test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	opts := &RawOptions{ConfigFile: cfgPath, PromptsFile: promptsPath}
	_, err := opts.Validate(context.Background())
	if err == nil {
		t.Error("expected error for invalid poll_interval")
	}
}

func TestValidate_Success(t *testing.T) {
	opts := &RawOptions{
		ConfigFile:  filepath.Join("..", "..", "config", "default_config.yaml"),
		PromptsFile: filepath.Join("..", "..", "config", "default_prompts.yaml"),
	}
	validated, err := opts.Validate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	completed, err := validated.Complete()
	if err != nil {
		t.Fatal(err)
	}
	if completed.Config.JiraProject != "ARO" {
		t.Errorf("expected jira_project ARO, got %s", completed.Config.JiraProject)
	}
}
