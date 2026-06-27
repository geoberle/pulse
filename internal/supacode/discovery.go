package supacode

import (
	"fmt"
	"os"
)

// Discover returns the Supacode Unix domain socket path
// from $SUPACODE_SOCKET_PATH. Pulse must run inside a Supacode terminal.
func Discover() (string, error) {
	if p := os.Getenv("SUPACODE_SOCKET_PATH"); len(p) != 0 {
		return p, nil
	}
	return "", fmt.Errorf("SUPACODE_SOCKET_PATH not set — run Pulse inside a Supacode terminal")
}
