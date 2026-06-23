package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidate_MissingConfigFile(t *testing.T) {
	opts := &RawOptions{
		ConfigFile:  "/nonexistent/config.yaml",
		PromptsFile: "/nonexistent/prompts.yaml",
	}
	_, err := opts.Validate()
	if err == nil {
		t.Error("expected error for missing config file")
	}
}

func TestValidate_NoRepos(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "config.yaml")
	promptsPath := filepath.Join(tmp, "prompts.yaml")

	if err := os.WriteFile(cfgPath, []byte("jira:\n  host: https://example.atlassian.net\njira_project: ARO\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(promptsPath, []byte("review_comment: test\nrebase: test\njira_update: test\njira_create: test\nci_failure: test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	opts := &RawOptions{ConfigFile: cfgPath, PromptsFile: promptsPath}
	_, err := opts.Validate()
	if err == nil {
		t.Error("expected error for no repos")
	}
}

func TestValidate_MissingJiraProject(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "config.yaml")
	promptsPath := filepath.Join(tmp, "prompts.yaml")

	if err := os.WriteFile(cfgPath, []byte("jira:\n  host: https://example.atlassian.net\nrepos:\n  - Azure/ARO-HCP\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(promptsPath, []byte("review_comment: test\nrebase: test\njira_update: test\njira_create: test\nci_failure: test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	opts := &RawOptions{ConfigFile: cfgPath, PromptsFile: promptsPath}
	_, err := opts.Validate()
	if err == nil {
		t.Error("expected error for missing jira_project")
	}
}

func TestValidate_MissingJiraHost(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "config.yaml")
	promptsPath := filepath.Join(tmp, "prompts.yaml")

	if err := os.WriteFile(cfgPath, []byte("repos:\n  - Azure/ARO-HCP\njira_project: ARO\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(promptsPath, []byte("review_comment: test\nrebase: test\njira_update: test\njira_create: test\nci_failure: test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	opts := &RawOptions{ConfigFile: cfgPath, PromptsFile: promptsPath}
	_, err := opts.Validate()
	if err == nil {
		t.Error("expected error for missing jira host")
	}
}

func TestValidate_HttpJiraHost(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "config.yaml")
	promptsPath := filepath.Join(tmp, "prompts.yaml")

	if err := os.WriteFile(cfgPath, []byte("jira:\n  host: http://example.atlassian.net\nrepos:\n  - Azure/ARO-HCP\njira_project: ARO\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(promptsPath, []byte("review_comment: test\nrebase: test\njira_update: test\njira_create: test\nci_failure: test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	opts := &RawOptions{ConfigFile: cfgPath, PromptsFile: promptsPath}
	_, err := opts.Validate()
	if err == nil {
		t.Error("expected error for http jira host")
	}
}

func TestValidate_InvalidPollInterval(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "config.yaml")
	promptsPath := filepath.Join(tmp, "prompts.yaml")

	if err := os.WriteFile(cfgPath, []byte("jira:\n  host: https://example.atlassian.net\nrepos:\n  - Azure/ARO-HCP\njira_project: ARO\npoll_interval: notaduration\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(promptsPath, []byte("review_comment: test\nrebase: test\njira_update: test\njira_create: test\nci_failure: test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	opts := &RawOptions{ConfigFile: cfgPath, PromptsFile: promptsPath}
	_, err := opts.Validate()
	if err == nil {
		t.Error("expected error for invalid poll_interval")
	}
}

func TestValidate_InvalidPromptTemplate(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "config.yaml")
	promptsPath := filepath.Join(tmp, "prompts.yaml")

	if err := os.WriteFile(cfgPath, []byte("jira:\n  host: https://example.atlassian.net\nrepos:\n  - Azure/ARO-HCP\njira_project: ARO\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(promptsPath, []byte("review_comment: \"{{.Broken\"\nrebase: test\njira_update: test\njira_create: test\nci_failure: test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	opts := &RawOptions{ConfigFile: cfgPath, PromptsFile: promptsPath}
	_, err := opts.Validate()
	if err == nil {
		t.Error("expected error for invalid prompt template")
	}
}

func TestValidate_Success(t *testing.T) {
	opts := &RawOptions{
		ConfigFile:  filepath.Join("..", "..", "config", "default_config.yaml"),
		PromptsFile: filepath.Join("..", "..", "config", "default_prompts.yaml"),
	}
	validated, err := opts.Validate()
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
