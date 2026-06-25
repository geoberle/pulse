package supacode

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// Discover returns the path to a live Supacode Unix domain socket.
// It checks $SUPACODE_SOCKET_PATH first, then scans /tmp/supacode-<uid>/pid-*
// and validates each candidate's PID is alive.
func Discover() (string, error) {
	if p := os.Getenv("SUPACODE_SOCKET_PATH"); len(p) != 0 {
		return p, nil
	}

	uid := os.Getuid()
	dir := fmt.Sprintf("/tmp/supacode-%d", uid)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("scan %s: %w", dir, err)
	}

	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, "pid-") {
			continue
		}
		pidStr := strings.TrimPrefix(name, "pid-")
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}
		if err := syscall.Kill(pid, 0); err == syscall.ESRCH {
			continue
		}
		return filepath.Join(dir, name), nil
	}

	return "", fmt.Errorf("no live Supacode socket found in %s", dir)
}
