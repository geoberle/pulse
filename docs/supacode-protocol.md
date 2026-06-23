# Supacode Domain Socket Protocol

Reference for the Supacode Unix domain socket protocol, used by Pulse's `internal/supacode/` client.

## Transport

- **Socket**: `AF_UNIX`, `SOCK_STREAM`
- **Path**: `/tmp/supacode-<uid>/pid-<pid>`
- **Max payload**: 64 KB
- **Read timeout**: 5 seconds

### Discovery

1. Inside Supacode terminal: `$SUPACODE_SOCKET_PATH`
2. Outside: scan `/tmp/supacode-<uid>/pid-*`, validate PID liveness with `kill(pid, 0)`

### Wire Protocol

Stateless request/response per connection:

1. Client opens Unix stream connection
2. Client writes UTF-8 JSON request
3. Client calls `shutdown(fd, SHUT_WR)` to signal end of request
4. Client reads response until EOF
5. Server closes FD after writing response

## Message Types

### Query (reads)

```json
{"query": "tabs", "worktreeID": "some-id"}
```
```json
{"ok": true, "data": [{"id": "tab-uuid", "focused": "1"}]}
```

| Resource | Required Params | Response Fields |
|---|---|---|
| `repos` | none | `id` |
| `worktrees` | none | `id`, `focused` |
| `tabs` | `worktreeID` | `id`, `focused` |
| `surfaces` | `worktreeID`, `tabID` | `id`, `focused` |
| `scripts` | `worktreeID` | `id`, `kind`, `name`, `displayName`, `running` |

### Command (mutations via deeplink)

```json
{"deeplink": "supacode://worktree/<id>/tab/new?input=echo%20hello"}
```
```json
{"ok": true}
```

**Worktree**: `worktree/<id>` (focus), `/run`, `/stop`, `/archive`, `/unarchive`, `/delete`, `/pin`, `/unpin`

**Tab**: `worktree/<id>/tab/new?input=<cmd>&id=<uuid>`, `tab/<tabID>` (focus), `tab/<tabID>/destroy`

**Surface**: `worktree/<id>/tab/<tabID>/surface/<surfID>/split?direction=h|v&input=<cmd>&id=<uuid>`, `surface/<surfID>` (focus, optional `?input=<cmd>`), `surface/<surfID>/destroy`

**Repo**: `repo/open?path=<path>`, `repo/<id>/worktree/new?branch=<br>&base=<base>&fetch=true`

## Notes

- All response values are strings (not typed)
- IDs: percent-encoded filesystem paths for repos/worktrees, UUIDs for tabs/surfaces
- Protocol is macOS-only but Go client needs no cgo
- Transport is ~100 lines: `net.Dial("unix")`, JSON marshal, `syscall.Shutdown(SHUT_WR)`, read until EOF
