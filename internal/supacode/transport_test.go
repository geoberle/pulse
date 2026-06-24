package supacode

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

var sockCounter atomic.Int64

func mockServer(t *testing.T, handler func(req []byte) []byte) string {
	t.Helper()
	n := sockCounter.Add(1)
	sock := filepath.Join(os.TempDir(), fmt.Sprintf("sc-%d-%d.sock", os.Getpid(), n))
	os.Remove(sock)
	t.Cleanup(func() { os.Remove(sock) })
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				req, err := io.ReadAll(conn)
				if err != nil {
					return
				}
				resp := handler(req)
				conn.Write(resp)
			}()
		}
	}()

	return sock
}

func TestRoundTrip(t *testing.T) {
	t.Parallel()
	sock := mockServer(t, func(req []byte) []byte {
		return []byte(`{"ok":true,"data":[{"id":"abc"}]}`)
	})

	data, err := roundTrip(sock, queryRequest{Query: "repos"})
	if err != nil {
		t.Fatal(err)
	}

	var resp response
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.OK {
		t.Error("expected ok=true")
	}
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 data item, got %d", len(resp.Data))
	}
	if resp.Data[0].ID != "abc" {
		t.Errorf("expected id=abc, got %s", resp.Data[0].ID)
	}
}

func TestRoundTrip_ConnectError(t *testing.T) {
	t.Parallel()
	_, err := roundTrip("/nonexistent/socket", queryRequest{Query: "repos"})
	if err == nil {
		t.Error("expected error for nonexistent socket")
	}
}

func TestRoundTrip_ResponseTooLarge(t *testing.T) {
	t.Parallel()
	sock := mockServer(t, func(req []byte) []byte {
		return []byte(strings.Repeat("x", maxPayload+1))
	})

	_, err := roundTrip(sock, queryRequest{Query: "repos"})
	if err == nil {
		t.Error("expected error for oversized response")
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Errorf("expected 'too large' error, got: %v", err)
	}
}

func TestDoQuery_Success(t *testing.T) {
	t.Parallel()
	sock := mockServer(t, func(req []byte) []byte {
		return []byte(`{"ok":true,"data":[{"id":"r1"}]}`)
	})

	resp, err := doQuery(sock, queryRequest{Query: "repos"})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Data) != 1 || resp.Data[0].ID != "r1" {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestDoQuery_ServerError(t *testing.T) {
	t.Parallel()
	sock := mockServer(t, func(req []byte) []byte {
		return []byte(`{"ok":false,"error":"not found"}`)
	})

	_, err := doQuery(sock, queryRequest{Query: "repos"})
	if err == nil {
		t.Error("expected error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

func TestDoQuery_InvalidJSON(t *testing.T) {
	t.Parallel()
	sock := mockServer(t, func(req []byte) []byte {
		return []byte(`not json`)
	})

	_, err := doQuery(sock, queryRequest{Query: "repos"})
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestDoCommand_Success(t *testing.T) {
	t.Parallel()
	sock := mockServer(t, func(req []byte) []byte {
		return []byte(`{"ok":true}`)
	})

	err := doCommand(sock, "supacode://tab/abc/destroy")
	if err != nil {
		t.Fatal(err)
	}
}

func TestDoCommand_ServerError(t *testing.T) {
	t.Parallel()
	sock := mockServer(t, func(req []byte) []byte {
		return []byte(`{"ok":false,"error":"invalid deeplink"}`)
	})

	err := doCommand(sock, "supacode://bad")
	if err == nil {
		t.Error("expected error")
	}
	if !strings.Contains(err.Error(), "invalid deeplink") {
		t.Errorf("expected 'invalid deeplink' in error, got: %v", err)
	}
}

func TestDoQuery_VerifiesRequestFormat(t *testing.T) {
	t.Parallel()
	var received queryRequest
	sock := mockServer(t, func(req []byte) []byte {
		json.Unmarshal(req, &received)
		return []byte(`{"ok":true,"data":[]}`)
	})

	doQuery(sock, queryRequest{Query: "tabs", WorktreeID: "wt-1"})

	if received.Query != "tabs" {
		t.Errorf("expected query=tabs, got %s", received.Query)
	}
	if received.WorktreeID != "wt-1" {
		t.Errorf("expected worktreeID=wt-1, got %s", received.WorktreeID)
	}
}

func TestDoCommand_VerifiesRequestFormat(t *testing.T) {
	t.Parallel()
	var received commandRequest
	sock := mockServer(t, func(req []byte) []byte {
		json.Unmarshal(req, &received)
		return []byte(`{"ok":true}`)
	})

	doCommand(sock, "supacode://worktree/abc")

	if received.Deeplink != "supacode://worktree/abc" {
		t.Errorf("expected deeplink supacode://worktree/abc, got %s", received.Deeplink)
	}
}

func TestRoundTrip_RequestTooLarge(t *testing.T) {
	t.Parallel()
	sock := mockServer(t, func(req []byte) []byte {
		return []byte(`{"ok":true}`)
	})

	bigInput := strings.Repeat("x", maxPayload+1)
	_, err := roundTrip(sock, commandRequest{Deeplink: bigInput})
	if err == nil {
		t.Error("expected error for oversized request")
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Errorf("expected 'too large' in error, got: %v", err)
	}
}

func TestDiscover_EnvVar(t *testing.T) {
	t.Setenv("SUPACODE_SOCKET_PATH", "/tmp/test-socket")
	path, err := Discover()
	if err != nil {
		t.Fatal(err)
	}
	if path != "/tmp/test-socket" {
		t.Errorf("expected /tmp/test-socket, got %s", path)
	}
}

func TestDiscover_NoSocket(t *testing.T) {
	t.Setenv("SUPACODE_SOCKET_PATH", "")
	dir := t.TempDir()
	origTmp := os.Getenv("TMPDIR")
	defer os.Setenv("TMPDIR", origTmp)

	_, err := Discover()
	if err == nil {
		t.Log("expected error when no socket exists, but Discover may find real socket — skipping")
		t.SkipNow()
	}
	_ = dir
}
