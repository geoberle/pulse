package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPrompts(t *testing.T) {
	t.Parallel()
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

func TestLoadPrompts_Errors(t *testing.T) {
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
			content: "review_comment: test\nunknown_field: oops\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var path string
			if len(tt.content) == 0 {
				path = "/nonexistent/prompts.yaml"
			} else {
				tmp := t.TempDir()
				path = filepath.Join(tmp, "prompts.yaml")
				if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
					t.Fatal(err)
				}
			}
			_, err := LoadPrompts(path)
			if err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestDefaultPromptsPath_XDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg-test")
	path, err := DefaultPromptsPath()
	if err != nil {
		t.Fatal(err)
	}
	if path != "/tmp/xdg-test/pulse/prompts.yaml" {
		t.Errorf("expected /tmp/xdg-test/pulse/prompts.yaml, got %s", path)
	}
}

func TestValidateTemplates(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		prompts Prompts
		wantErr bool
	}{
		{
			name: "valid defaults",
			prompts: Prompts{
				ReviewComment: "Review {{.File}}",
				Rebase:        "Rebase {{.Branch}}",
				JiraUpdate:    "Update {{.Key}}",
				JiraCreate:    "Create {{.Title}}",
				CIFailure:     "CI failed {{.Check}}",
			},
		},
		{
			name: "empty prompt",
			prompts: Prompts{
				ReviewComment: "",
				Rebase:        "test",
				JiraUpdate:    "test",
				JiraCreate:    "test",
				CIFailure:     "test",
			},
			wantErr: true,
		},
		{
			name: "invalid syntax",
			prompts: Prompts{
				ReviewComment: "{{.Broken",
				Rebase:        "test",
				JiraUpdate:    "test",
				JiraCreate:    "test",
				CIFailure:     "test",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.prompts.ValidateTemplates()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTemplates() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
