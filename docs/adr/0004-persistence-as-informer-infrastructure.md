# Persistence as informer infrastructure, not a handler

State persistence (writing the WorkItem tree to disk) is part of the informer's poll cycle, not a registered handler. The informer calls `store.Save()` after updating its cache but before dispatching events. On startup, `store.Load()` seeds the cache so the first poll diffs against persisted state rather than an empty tree.

## Considered Options

**Handler-based persistence** — a `State` handler registered via `RegisterHandler()`, like all other consumers. On each event, it reads the full cache via a callback or Lister and serializes to disk. Follows the existing handler pattern — no special treatment. Rejected because handlers fire per-event (not per-poll), so a single poll producing 15 events means 15 redundant full-tree writes. A post-poll lifecycle hook was considered to fix this, but adding a second dispatch mechanism just for persistence signals that persistence is a different concern than event handling. More importantly, handlers fire after cache update — if persistence is a handler, the ordering `cache update → dispatch → persist` means a crash between dispatch and persist loses state. Handlers that use a Lister would see cache state that isn't yet on disk.

**Informer-internal persistence** (chosen) — the informer owns an injected `Store` interface with `Save()` and `Load()`. The poll cycle becomes: `source.List()` → `diffTrees()` → update cache → `store.Save()` → `dispatch()`. Persistence happens exactly once per poll, at the right moment. On startup, `store.Load()` returns the persisted tree, the informer uses it as initial cache, and the first poll diffs against it — producing only events for what actually changed while the process was down. The Store is injected (not hardcoded) so tests can use an in-memory implementation.

## Consequences

- On successful Save, disk matches cache when handlers fire. Handlers using a Lister see state that is both current and persisted. Save failures are non-fatal — state is a cache, pollers hold ground truth.
- Persistence is exactly once per poll cycle, not once per event. No wasted writes.
- `Store` is an interface in the informer package, keeping the informer testable without disk I/O. The JSON file implementation lives in a separate package.
- Startup is seamless: load from disk, diff on first poll, only real changes produce events. The TUI renders instantly from loaded state before the first network call completes.
- Save failures are logged but don't block event dispatch — state is a cache, pollers hold ground truth.
