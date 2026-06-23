package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"gopkg.in/yaml.v3"
)

type Prompts struct {
	ReviewComment PromptTemplate `yaml:"review_comment"`
	Rebase        PromptTemplate `yaml:"rebase"`
	JiraUpdate    PromptTemplate `yaml:"jira_update"`
	JiraCreate    PromptTemplate `yaml:"jira_create"`
	CIFailure     PromptTemplate `yaml:"ci_failure"`
}

type PromptTemplate struct {
	Prompt string `yaml:"prompt"`
}

func LoadPrompts(path string) (*Prompts, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read prompts %s: %w", path, err)
	}
	p := &Prompts{}
	if err := yaml.Unmarshal(data, p); err != nil {
		return nil, fmt.Errorf("parse prompts %s: %w", path, err)
	}
	return p, nil
}

func Render(tmpl string, data any) (string, error) {
	t, err := template.New("prompt").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}
	return buf.String(), nil
}

func DefaultPromptsPath() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); len(xdg) > 0 {
		return filepath.Join(xdg, "pulse", "prompts.yaml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determine home directory: %w", err)
	}
	return filepath.Join(home, ".config", "pulse", "prompts.yaml"), nil
}
