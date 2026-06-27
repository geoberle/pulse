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
