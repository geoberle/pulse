package config

import (
	"path/filepath"
	"testing"
)

func TestLoadPrompts(t *testing.T) {
	prompts, err := LoadPrompts(filepath.Join("..", "..", "config", "default_prompts.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	if len(prompts.ReviewComment.Prompt) == 0 {
		t.Error("expected non-empty review_comment prompt")
	}
	if len(prompts.Rebase.Prompt) == 0 {
		t.Error("expected non-empty rebase prompt")
	}
	if len(prompts.JiraUpdate.Prompt) == 0 {
		t.Error("expected non-empty jira_update prompt")
	}
	if len(prompts.JiraCreate.Prompt) == 0 {
		t.Error("expected non-empty jira_create prompt")
	}
	if len(prompts.CIFailure.Prompt) == 0 {
		t.Error("expected non-empty ci_failure prompt")
	}
}

func TestRender(t *testing.T) {
	tests := []struct {
		name     string
		tmpl     string
		data     any
		expected string
	}{
		{
			name: "review comment",
			tmpl: `Review comment on {{.File}} in PR #{{.PRNumber}}`,
			data: map[string]any{
				"File":     "constants.go",
				"PRNumber": 891,
			},
			expected: "Review comment on constants.go in PR #891",
		},
		{
			name: "rebase",
			tmpl: `Rebase {{.Branch}} onto {{.TargetBranch}}`,
			data: map[string]any{
				"Branch":       "feature/ARO-12345-dns",
				"TargetBranch": "main",
			},
			expected: "Rebase feature/ARO-12345-dns onto main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Render(tt.tmpl, tt.data)
			if err != nil {
				t.Fatal(err)
			}
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestRender_InvalidTemplate(t *testing.T) {
	_, err := Render("{{.Broken", nil)
	if err == nil {
		t.Error("expected error for invalid template")
	}
}
