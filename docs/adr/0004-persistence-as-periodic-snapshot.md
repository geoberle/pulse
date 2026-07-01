# Persistence as periodic cache snapshot

State persistence (writing the WorkItem tree to disk) is a periodic goroutine that snapshots the informer cache at pollInterval via `Lister → BuildTree → Store.Save`. On startup, `Store.Load()` provides pre-sync rendering data independent of the informer cache.

## Context

The application uses `cache.SharedIndexInformer` from client-go, which manages its own cache and event distribution internally. There is no hook between "cache updated" and "handlers fired" to insert a synchronous save. Persistence is therefore decoupled from the informer's event loop.

## Considered Options

**Handler-based persistence** — a handler registered via `AddEventHandler()` that serializes on each event. Rejected: handlers fire per-event (not per-poll), so a single poll producing 15 events means 15 redundant full-tree writes. Adding a post-poll lifecycle hook was considered, but the extra mechanism signals persistence is a different concern than event handling.

**Informer-internal persistence** (persist-before-dispatch) — the informer owns an injected Store and calls `Save()` between cache update and event dispatch, guaranteeing disk matches cache when handlers fire. Rejected: SharedIndexInformer provides no insertion point between cache update and event dispatch. Reimplementing informer internals to add one would forfeit client-go's thread-safe caching, indexing, event distribution, resync, and workqueue readiness.

## Decision

**Periodic decoupled snapshots** — a goroutine calls `Lister.List() → BuildTree → Store.Save()` every pollInterval. BuildTree DeepCopies items from the Lister before mutation, preventing races with the reflector's cache updates. On startup, `cachedSource` reads `Store.Load()` on the first List call to seed the informer cache — the reflector diffs against this seeded state and only emits events for real changes.

## Consequences

- Save failures are non-fatal — state is a cache, pollers hold ground truth.
- Persistence runs on an independent ticker at pollInterval, not synchronized with the informer's relist cycle. Disk may lag cache by up to one poll interval.
- `Store` is standalone (not injected into the informer), keeping the informer testable without disk I/O. The JSON file implementation lives in a separate package.
- Startup is seamless: load from disk for instant TUI, informer catches up in background.
- Disk may lag cache by up to one poll interval. Acceptable: a crash loses at most one cycle of changes, recovered on next live poll.
