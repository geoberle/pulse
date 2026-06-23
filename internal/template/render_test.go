package template

import (
	"testing"
)

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

func TestRender_MissingKey(t *testing.T) {
	_, err := Render("Hello {{.Missing}}", map[string]any{"Other": "value"})
	if err == nil {
		t.Error("expected error for missing template key")
	}
}

func TestRender_InvalidTemplate(t *testing.T) {
	_, err := Render("{{.Broken", nil)
	if err == nil {
		t.Error("expected error for invalid template")
	}
}

func TestValidateSyntax(t *testing.T) {
	if err := ValidateSyntax("good", "Hello {{.Name}}"); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestValidateSyntax_Invalid(t *testing.T) {
	if err := ValidateSyntax("bad", "{{.Broken"); err == nil {
		t.Error("expected error for invalid template syntax")
	}
}
