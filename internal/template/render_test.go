package template

import (
	"testing"
)

func TestRender(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		tmpl     string
		data     any
		expected string
		wantErr  bool
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
		{
			name:    "missing key",
			tmpl:    "Hello {{.Missing}}",
			data:    map[string]any{"Other": "value"},
			wantErr: true,
		},
		{
			name:    "invalid template",
			tmpl:    "{{.Broken",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := Render(tt.name, tt.tmpl, tt.data)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Render() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestValidateSyntax(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		tmpl    string
		wantErr bool
	}{
		{
			name: "valid",
			tmpl: "Hello {{.Name}}",
		},
		{
			name:    "invalid",
			tmpl:    "{{.Broken",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateSyntax(tt.name, tt.tmpl)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSyntax() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
