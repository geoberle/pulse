package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPrompts(t *testing.T) {
	prompts, err := LoadPrompts(filepath.Join("..", "..", "config", "default_prompts.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	if len(prompts.ReviewComment) == 0 {
		t.Error("expected non-empty review_comment prompt")
	}
	if len(prompts.Rebase) == 0 {
		t.Error("expected non-empty rebase prompt")
	}
	if len(prompts.JiraUpdate) == 0 {
		t.Error("expected non-empty jira_update prompt")
	}
	if len(prompts.JiraCreate) == 0 {
		t.Error("expected non-empty jira_create prompt")
	}
	if len(prompts.CIFailure) == 0 {
		t.Error("expected non-empty ci_failure prompt")
	}
}

func TestLoadPrompts_MissingFile(t *testing.T) {
	_, err := LoadPrompts("/nonexistent/prompts.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoadPrompts_InvalidYAML(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "prompts.yaml")
	if err := os.WriteFile(path, []byte(":\ninvalid: [yaml\n"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadPrompts(path)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestValidateTemplates(t *testing.T) {
	prompts, err := LoadPrompts(filepath.Join("..", "..", "config", "default_prompts.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := prompts.ValidateTemplates(); err != nil {
		t.Errorf("expected default prompts to validate, got %v", err)
	}
}

func TestValidateTemplates_EmptyPrompt(t *testing.T) {
	p := &Prompts{
		ReviewComment: "",
		Rebase:        "test",
		JiraUpdate:    "test",
		JiraCreate:    "test",
		CIFailure:     "test",
	}
	if err := p.ValidateTemplates(); err == nil {
		t.Error("expected error for empty prompt")
	}
}

func TestValidateTemplates_InvalidSyntax(t *testing.T) {
	p := &Prompts{
		ReviewComment: "{{.Broken",
		Rebase:        "test",
		JiraUpdate:    "test",
		JiraCreate:    "test",
		CIFailure:     "test",
	}
	if err := p.ValidateTemplates(); err == nil {
		t.Error("expected error for invalid template syntax")
	}
}
