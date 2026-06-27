package github

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Token extracts a GitHub API token by running "gh auth token".
func Token(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "gh", "auth", "token").Output()
	if err != nil {
		return "", fmt.Errorf("gh auth token: %w", err)
	}
	token := strings.TrimSpace(string(out))
	if len(token) == 0 {
		return "", fmt.Errorf("gh auth token returned empty")
	}
	return token, nil
}

// User returns the authenticated GitHub username by running "gh api user".
func User(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "gh", "api", "user", "--jq", ".login").Output()
	if err != nil {
		return "", fmt.Errorf("gh api user: %w", err)
	}
	login := strings.TrimSpace(string(out))
	if len(login) == 0 {
		return "", fmt.Errorf("gh api user returned empty login")
	}
	return login, nil
}
