package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidate(t *testing.T) {
	t.Parallel()

	validConfig := "jira:\n  host: https://example.atlassian.net\nrepos:\n  - Azure/ARO-HCP\njira_project: ARO\n"
	validPrompts := "review_comment: test\nrebase: test\njira_update: test\njira_create: test\nci_failure: test\n"

	tests := []struct {
		name    string
		config  string
		prompts string
		wantErr bool
	}{
		{
			name:    "missing config file",
			config:  "",
			prompts: "",
			wantErr: true,
		},
		{
			name:    "no repos",
			config:  "jira:\n  host: https://example.atlassian.net\njira_project: ARO\n",
			prompts: validPrompts,
			wantErr: true,
		},
		{
			name:    "missing jira_project",
			config:  "jira:\n  host: https://example.atlassian.net\nrepos:\n  - Azure/ARO-HCP\n",
			prompts: validPrompts,
			wantErr: true,
		},
		{
			name:    "missing jira host",
			config:  "repos:\n  - Azure/ARO-HCP\njira_project: ARO\n",
			prompts: validPrompts,
			wantErr: true,
		},
		{
			name:    "http jira host",
			config:  "jira:\n  host: http://example.atlassian.net\nrepos:\n  - Azure/ARO-HCP\njira_project: ARO\n",
			prompts: validPrompts,
			wantErr: true,
		},
		{
			name:    "invalid poll_interval",
			config:  "jira:\n  host: https://example.atlassian.net\nrepos:\n  - Azure/ARO-HCP\njira_project: ARO\npoll_interval: notaduration\n",
			prompts: validPrompts,
			wantErr: true,
		},
		{
			name:    "invalid prompt template",
			config:  validConfig,
			prompts: "review_comment: \"{{.Broken\"\nrebase: test\njira_update: test\njira_create: test\nci_failure: test\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if len(tt.config) == 0 {
				opts := &RawOptions{
					ConfigFile:  "/nonexistent/config.yaml",
					PromptsFile: "/nonexistent/prompts.yaml",
				}
				_, err := opts.Validate()
				if err == nil {
					t.Error("expected error")
				}
				return
			}

			tmp := t.TempDir()
			cfgPath := filepath.Join(tmp, "config.yaml")
			promptsPath := filepath.Join(tmp, "prompts.yaml")

			if err := os.WriteFile(cfgPath, []byte(tt.config), 0644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(promptsPath, []byte(tt.prompts), 0644); err != nil {
				t.Fatal(err)
			}

			opts := &RawOptions{ConfigFile: cfgPath, PromptsFile: promptsPath}
			_, err := opts.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidate_Success(t *testing.T) {
	t.Parallel()
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
