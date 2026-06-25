package supacode

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"syscall"
	"time"
)

const (
	maxPayload = 64 * 1024
	ioTimeout  = 5 * time.Second
)

// roundTrip performs one request/response cycle over a Unix domain socket.
// It connects, writes the JSON request, signals end-of-write via
// shutdown(SHUT_WR), then reads the full response until EOF.
func roundTrip(socketPath string, req any) ([]byte, error) {
	conn, err := net.DialTimeout("unix", socketPath, ioTimeout)
	if err != nil {
		return nil, fmt.Errorf("connect %s: %w", socketPath, err)
	}
	defer func() { _ = conn.Close() }()

	if err := conn.SetDeadline(time.Now().Add(ioTimeout)); err != nil {
		return nil, fmt.Errorf("set deadline: %w", err)
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	if len(data) > maxPayload {
		return nil, fmt.Errorf("request too large: %d bytes (max %d)", len(data), maxPayload)
	}

	if _, err := conn.Write(data); err != nil {
		return nil, fmt.Errorf("write request: %w", err)
	}

	raw, ok := conn.(*net.UnixConn)
	if !ok {
		return nil, fmt.Errorf("expected *net.UnixConn, got %T", conn)
	}
	sc, err := raw.SyscallConn()
	if err != nil {
		return nil, fmt.Errorf("get syscall conn: %w", err)
	}
	var shutdownErr error
	if err := sc.Control(func(fd uintptr) {
		shutdownErr = syscall.Shutdown(int(fd), syscall.SHUT_WR)
	}); err != nil {
		return nil, fmt.Errorf("control fd: %w", err)
	}
	if shutdownErr != nil {
		return nil, fmt.Errorf("shutdown(SHUT_WR): %w", shutdownErr)
	}

	resp, err := io.ReadAll(io.LimitReader(conn, maxPayload+1))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if len(resp) > maxPayload {
		return nil, fmt.Errorf("response too large: >%d bytes", maxPayload)
	}

	return resp, nil
}

// doQuery sends a query request and returns the parsed response.
func doQuery(socketPath string, req queryRequest) (*response, error) {
	data, err := roundTrip(socketPath, req)
	if err != nil {
		return nil, err
	}
	var resp response
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	if !resp.OK {
		if len(resp.Err) > 0 {
			return nil, fmt.Errorf("query %q: %s", req.Query, resp.Err)
		}
		return nil, fmt.Errorf("query %q failed", req.Query)
	}
	return &resp, nil
}

// doCommand sends a deeplink command and validates the response.
func doCommand(socketPath string, deeplink string) error {
	data, err := roundTrip(socketPath, commandRequest{Deeplink: deeplink})
	if err != nil {
		return err
	}
	var resp response
	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}
	if !resp.OK {
		if len(resp.Err) > 0 {
			return fmt.Errorf("command %q: %s", deeplink, resp.Err)
		}
		return fmt.Errorf("command %q failed", deeplink)
	}
	return nil
}
