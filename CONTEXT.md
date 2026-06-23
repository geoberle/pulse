# Pulse — Domain Language

## Glossary

- **Work Item**: Top-level unit in the dashboard. Always a Jira issue (Story or Bug in the ARO project). The primary axis of organization. A work item has 0+ linked PRs across Azure/ARO-HCP and Azure/ARO-Tools.
- **Orphan PR**: A PR authored by the user that has no Jira link. Surfaced separately in the TUI and requires attention — either link to an existing work item or create a new Jira issue for it.
- **Local Work**: A worktree/branch with no PR yet. Detected via `supacode worktree list`. Linked to a Jira issue if the branch name contains a Jira key, otherwise shown as a top-level item.
- **Action**: Something on a work item or its PRs that needs attention. Examples: failed CI, review comment, needs rebase, stale Jira. Actions have an autonomy tier (propose, judgment).
- **Autonomy Tier**: How much human involvement an action requires. Propose = tool drafts, user approves inline. Judgment = opens a Supacode split with Claude. Auto tier designed but not enabled in v1 — all actions require user involvement.
- **Informer**: Central component that holds the canonical WorkItem tree, diffs against previous state on each poll, and fires OnAdd/OnUpdate/OnDelete events to registered handlers. K8s SharedInformer-style.
- **Engine**: Orchestrator that owns the poll loop, calls pollers, assembles the tree from flat poller results, and feeds the informer. Also owns the split-close watcher.
- **Poller**: Source-specific data fetcher. Returns flat list of WorkItems. Engine assembles them into the tree.
- **Handler**: Consumer registered with the informer. Receives events and performs side effects (TUI rendering, state persistence, LLM summarization, split lifecycle).

## Decisions

### Domain

- Jira issue types in scope: Story, Bug. Not Task (reserved for future agent-driven sub-work under stories/bugs).
- Jira project: ARO. Filtered to issues created by or assigned to the user.
- GitHub repos: Azure/ARO-HCP, Azure/ARO-Tools.
- PR-to-Jira linkage: regex `ARO-\d+` against branch name → PR title → PR body. First match wins.
- Review comments: no distinction between copilot and human reviewers. All unresolved comments where user hasn't replied = needs attention. Handled = user replied or conversation resolved.
- Copilot reviews come from `copilot-pull-request-reviewer[bot]` (standard PR review API, no special endpoint).
- Three top-level item types: Jira work item, Orphan PR, Local Work.

### Data Model

- Unified WorkItem model: common metadata (TypeMeta + ObjectMeta) with type-specific `Spec` as `json.RawMessage`, dispatched to typed structs on unmarshal. K8s-style pattern. Recursive via `Children []*WorkItem`. See ADR-0002.
- Kinds: `jira`, `pr`, `check`, `review`, `local`. `Kind` is a typed `string` enum with `Validate()`, not a bare string.
- All domain enums (`Kind`, `StalenessState`, `BranchState`) follow the same pattern: typed string, constants with zero-value = unknown/unset, `Validate()` method that guards against invalid values.
- All Spec types validate required fields on unmarshal (e.g. `JiraSpec.Key`, `PRSpec.Repo/Number/Branch`).
- IDs follow `{source}:{identifier}` pattern (e.g. `jira:ARO-12345`, `pr:Azure/ARO-HCP:891`, `gh-comment:3453365398`).

### TUI

- Tree is always fully expanded. No collapse/expand.
- Items auto-remove when fully resolved upstream. Zero-noise principle.
- Cursor navigates every line (Jira row, PR row, action row).
- Keybindings: `enter` = action menu (always, even if empty). `o` = open upstream in browser. `s` = switch to active Claude session. `R` = force refresh. `j`/`k` or `↑`/`↓` = navigate. `q` = quit. `?` = help.
- Action menus are context-sensitive per item type and level.
- No dismiss feature. Items stay until resolved upstream.
- Propose actions: inline y/n for simple single-command actions.
- Judgment actions: open Supacode horizontal split with Claude.
- Status bar: last poll time, next poll time, total action count. Errors shown inline.
- Startup: load persisted state → render instantly → show local worktrees (no network) → backfill GH + Jira async → diff + mark new items.
- Review comment display: AI summary ~40 chars + file path. Skip summarization if body ≤40 chars.
- CI check display: check name only, no log snippets.
- Bubbletea: channel-based event subscription from informer. No manual goroutines. Cmd-based async pattern.

### Supacode Integration

- Split target: match branch to existing worktree → open there. No worktree → main worktree, Claude gets context via prompt.
- Split-close detection: poll `supacode surface list` for surface ID. When gone → re-fetch upstream state immediately. Fallback: regular 5 min poll.
- Split dedup: TUI tracks surface ID per action. `enter` on existing split → focuses it instead of opening new one.
- TUI and Claude split share no state. TUI infers results by inspecting upstream.

### Prompt Templates

- No custom spoke skills. Claude sessions launched with user-configurable prompt templates (`~/.config/pulse/prompts.yaml`).
- Go template syntax with action-type-specific variables (e.g. `{{.PRNumber}}`, `{{.CommentBody}}`, `{{.IssueKey}}`).
- Ship with sensible defaults, user overrides.
- TUI is decoupled from Claude's skill ecosystem. Users reference whatever skills/agents they want in their templates.

### Architecture

- Informer pattern: pollers → engine (assembles tree) → informer (diffs, dispatches events) → handlers. See ADR-0001.
- Handlers: TUI (channel → bubbletea Cmd), State (persists state.json), Summarizer (LLM on new/changed reviews), Split watcher (prunes dead splits on delete).
- Pollers return flat lists, engine assembles tree (PR-to-Jira matching by regex, orphan detection, local work grouping).
- Partial poll failure → use succeeded sources, show error indicator for failed ones.
- Informer diff: hash of Kind + ID + Label + Status + Spec bytes. Children diffed separately (recursive). No event bubbling from child to parent.
- Engine owns poll loop (5 min interval), split-close watcher goroutine, manual refresh channel.
- Cobra CLI with RawOptions → ValidatedOptions → CompletedOptions pattern (from ARO-HCP templatize).

### Config Conventions

- YAML library: `sigs.k8s.io/yaml` with `UnmarshalStrict`. Struct tags are `json:` (not `yaml:`) because `sigs.k8s.io/yaml` delegates to `encoding/json`. Unknown keys in config files are rejected at load time.
- Time-related config fields use Go duration strings (e.g. `"120h"`, `"5m"`), never integer-unit fields (e.g. `days: 5`). Parsed via `time.ParseDuration`. This applies to all current and future time config.

### Dependencies

- **TUI**: bubbletea (Charm)
- **CLI**: cobra
- **YAML**: `sigs.k8s.io/yaml` (K8s ecosystem, strict mode)
- **GitHub**: `google/go-github` SDK, token from `gh auth token`
- **Jira**: `ctreminiom/go-atlassian` SDK, PAT from config
- **LLM**: `anthropics/anthropic-sdk-go` via Vertex AI (Haiku)
- **Supacode**: native Go client over Unix domain socket. Queries for reads, deeplink commands for mutations. No CLI dependency. Protocol documented in `docs/supacode-protocol.md`.

### Storage

- Config: `~/.config/pulse/config.yaml` + `prompts.yaml`. XDG-compliant.
- State: `~/.local/state/pulse/state.json`. Entire WorkItem tree persisted on every poll. Written atomically (tmp + rename).
- State includes: comment summary cache (by comment ID + body hash), split-to-action mappings (pruned on startup against `supacode surface list`).
- In-memory cache: PR data, Jira data, worktree list. Rebuilt every poll cycle.

### Testing

- Golden fixtures for informer diff tests (input trees → expected events).
- Golden fixtures for engine merge tests (poller results → expected tree).
- Pollers tested with API mocks.
- TUI tests skipped for v1.
