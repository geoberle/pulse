package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	tmpl "github.com/geoberle/pulse/internal/template"
)

// Prompts holds user-configurable Go template strings for each action type.
// Templates use Go text/template syntax with action-specific variables
// (e.g. {{.PRNumber}}, {{.CommentBody}}).
type Prompts struct {
	// ReviewComment is the template rendered when opening a Claude split
	// for an unresolved review comment.
	ReviewComment string `yaml:"review_comment"`

	// Rebase is the template rendered when proposing a rebase action.
	Rebase string `yaml:"rebase"`

	// JiraUpdate is the template rendered when updating a Jira issue
	// with current PR state.
	JiraUpdate string `yaml:"jira_update"`

	// JiraCreate is the template rendered when creating a new Jira issue
	// for an orphan PR.
	JiraCreate string `yaml:"jira_create"`

	// CIFailure is the template rendered when diagnosing a failed CI check.
	CIFailure string `yaml:"ci_failure"`
}

// LoadPrompts reads and parses a prompts YAML file.
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

// ValidateTemplates checks that all prompt templates are non-empty and
// syntactically valid Go templates. Call at startup to catch errors early
// rather than mid-session.
func (p *Prompts) ValidateTemplates() error {
	templates := []struct {
		name string
		body string
	}{
		{"review_comment", p.ReviewComment},
		{"rebase", p.Rebase},
		{"jira_update", p.JiraUpdate},
		{"jira_create", p.JiraCreate},
		{"ci_failure", p.CIFailure},
	}
	for _, t := range templates {
		if len(t.body) == 0 {
			return fmt.Errorf("prompt %q is empty", t.name)
		}
		if err := tmpl.ValidateSyntax(t.name, t.body); err != nil {
			return err
		}
	}
	return nil
}

// DefaultPromptsPath returns the XDG-compliant default path for the
// prompts configuration file.
func DefaultPromptsPath() (string, error) {
	dir, err := defaultConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "prompts.yaml"), nil
}
