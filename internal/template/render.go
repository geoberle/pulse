package template

import (
	"bytes"
	"fmt"
	"text/template"
)

// Render executes a Go text/template against data and returns the result.
// Uses missingkey=error to fail on undefined template variables rather
// than silently rendering "<no value>".
func Render(name, tmpl string, data any) (string, error) {
	t, err := template.New(name).Option("missingkey=error").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("parse template %s: %w", name, err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template %s: %w", name, err)
	}
	return buf.String(), nil
}

// ValidateSyntax checks that a Go template string is syntactically valid.
// Does not execute the template — only verifies it can be parsed.
func ValidateSyntax(name, tmpl string) error {
	_, err := template.New(name).Option("missingkey=error").Parse(tmpl)
	if err != nil {
		return fmt.Errorf("template %s: %w", name, err)
	}
	return nil
}
