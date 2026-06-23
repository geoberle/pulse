# Informer pattern for data flow

Pulse needs to propagate upstream state changes (GitHub, Jira, Supacode) to multiple consumers: the TUI renderer, the state persistence layer, the LLM summarizer, and the split watcher. We use a K8s SharedInformer-style pattern: pollers feed a complete WorkItem tree into a central informer, which diffs against its cache and fires OnAdd/OnUpdate/OnDelete events to registered handlers. This decouples all producers from all consumers via a single event interface.

## Considered Options

**Direct coupling** — pollers write directly into the bubbletea model, persistence and summarization happen inline in the TUI's Update function. Simpler, fewer abstractions, works fine for a single consumer. Rejected because adding a second consumer (e.g. summarizer) means threading it through TUI code that shouldn't know about LLM calls. Testing any consumer requires standing up the full TUI.

**Channel-per-consumer** — pollers send raw data on typed channels, each consumer reads its own channel. Avoids the shared cache but duplicates diff logic across consumers — each one has to figure out what changed independently. Also requires careful channel lifecycle management.

**Informer with event handlers** (chosen) — one shared cache, one diff, N handlers. Consumers implement a single `OnEvent(Event)` method. Adding a new consumer is one struct. Testing a consumer means feeding it synthetic events — no TUI, no pollers, no network. The diff is recursive (tree-structured WorkItems with children) and runs once per poll cycle regardless of consumer count.

## Consequences

- Every consumer sees a consistent sequence of events derived from the same diff. No divergence between what the TUI shows and what gets persisted.
- The informer holds the canonical state. Consumers don't cache separately — they react.
- Polling frequency and event granularity are coupled: a 5-minute poll means at most 5-minute event latency. This is fine for our use case but would need a watch stream for real-time needs.
