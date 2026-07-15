# Pulse — Product Requirements Document

## Overview

Pulse is a developer workflow orchestration tool that ties together Jira, GitHub, and local git worktrees into a single system. It ensures every piece of work has a Jira ticket, surfaces items that need attention, and uses AI to reduce the friction of Jira maintenance.

Pulse runs as a local server daemon with a TUI client. A future iOS client connects to the same API.

## Problem

- Jira tickets are frequently missing for active work (PRs without tickets, worktrees without tickets).
- No single view shows all active work across repos with their CI, review, and Jira status.
- Context switches between GitHub, Jira, and local repos waste time.
- Post-merge cleanup is forgotten — stale worktrees, Jira tickets left open.
- Jira ticket creation is tedious enough to skip.

## Users

Single user: the developer running Pulse locally. All data is personal — authored PRs, assigned/created Jira tickets, local worktrees.

## Core Concepts

### Work Item

The top-level entity. Always maps 1:1 to a Jira ticket. A work item groups one or more worktrees and pull requests under a single Jira ticket. Completion requires all PRs merged, all worktrees removed, and Jira closed.

### Untracked Worktree / Untracked PR

A worktree or PR that has no Jira link. Surfaced in "Needs Attention" until the user links it to an existing Jira ticket or creates a new one.

### Attention Item

Something on a work item that requires user action. Attention items drive the TUI's priority view.

## Architecture

### Single Binary

```
pulse serve     # start daemon
pulse tui       # connect TUI to running daemon
pulse status    # quick CLI status check
pulse auth login # Jira OAuth flow
```

One Go binary, Cobra subcommands. Server started manually (`pulse serve` in tmux or background).

### Server Daemon

Single process. Goroutines handle:
- Per-source pollers (git, GitHub, Jira) on independent intervals
- REST API handlers
- WebSocket event broadcasting
- AI task execution

### Communication

- **REST** for commands (create Jira, link worktree, force refresh)
- **WebSocket** for live updates (TUI subscribes, server pushes state changes)

### Storage

**SQLite** as a durable cache. External APIs (GitHub, Jira) and local git are the sources of truth. SQLite stores:
- Raw poller data (per-poller tables: `jira_tickets`, `pull_requests`, `worktrees`)
- Manual links (`manual_links` table for explicit worktree/PR-to-Jira overrides)
- AI draft cache

Schema is hybrid-normalized: each poller owns its table, engine assembles the tree in memory at query time.

### Authentication

| Service | Method |
|---------|--------|
| Jira | OAuth 2.0 with DCR + PKCE, browser-based login, auto-refresh via `TokenSource`. Tokens stored in `~/.local/state/pulse/jira_tokens.json` (atomic write, 0600). Port from Pulse v1 `jira-auth` branch. |
| GitHub | Delegate to `gh auth token`. No token stored by Pulse. |
| Vertex AI | Google Application Default Credentials (ADC). Config specifies project + region. |

### Configuration

`~/.config/pulse/config.yaml` (XDG-compliant):

```yaml
repos:
  - owner: Azure
    name: ARO-HCP
    path: ~/dev/aro-hcp
  - owner: Azure
    name: ARO-Tools
    path: ~/dev/ARO-Tools

jira:
  host: https://redhat.atlassian.net
  project: ARO
  component: aro-hcp-1p
  default_issue_type: Story

llm:
  provider: vertex
  project: my-gcp-project
  region: us-east5

poll_intervals:
  git: 30s
  github: 5m
  jira: 5m

stale_threshold: 120h
```

YAML library: `sigs.k8s.io/yaml` with `UnmarshalStrict`, `json:` struct tags.

## Data Model

### Entities

```
WorkItem (= Jira ticket)
├── key: ARO-12345
├── summary, description, status, type, epic_key
├── state: derived (planning | in-progress | in-review | merged | done)
├── attention_reasons: []
│
├── Worktree (child, 0..N)
│   ├── repo: Azure/ARO-HCP
│   ├── branch: fleet-caching
│   ├── path: ~/dev/aro-hcp.fleet-caching
│   └── has_commits: bool
│
└── PullRequest (child, 0..N)
    ├── repo, number, branch, url
    ├── status: open | merged | closed
    ├── ci_status: passing | failing | pending
    ├── review_status: approved | changes_requested | pending
    └── unresolved_comments: int

UntrackedWorktree
├── repo, branch, path
├── has_commits: bool
├── claude_session: bool
└── suggested_jira: []

UntrackedPR
├── repo, number, branch, url
├── ci_status, review_status
└── suggested_jira: []
```

### Relationships

- WorkItem 1→N Worktrees
- WorkItem 1→N PullRequests
- Worktree 1→0..1 PullRequest (matched by branch name)
- UntrackedWorktree / UntrackedPR → promoted to WorkItem child on link

### Linking Logic

Jira key detection (first match wins):
1. Branch name matches `ARO-\d+`
2. PR title contains `ARO-\d+`
3. PR body contains `ARO-\d+`
4. Manual link in SQLite `manual_links` table

Worktree↔PR matching: branch name equality across same repo.

### Derived State

No internal state machine. State is computed from external reality:

| State | Condition |
|-------|-----------|
| `planning` | Worktree(s) exist, no commits |
| `in-progress` | Commits exist, no PR |
| `in-review` | At least one PR open |
| `merged` | All PRs merged, worktree or Jira still open |
| `done` | All PRs merged + all worktrees removed + Jira closed |

## Pollers

### Git Poller
- Runs `git worktree list` for each configured repo
- Returns: branch, path, commit status (has commits beyond main)
- Interval: configurable, default 30s
- Local only, no API calls

### GitHub Poller
- Fetches PRs authored by user for each configured repo
- Includes: CI checks (GitHub checks API), review status, unresolved comments
- Auth: `gh auth token`
- Interval: configurable, default 5m

### Jira Poller
- Fetches tickets: project ARO, component aro-hcp-1p, assigned to or created by user
- Issue types: Story, Bug
- Includes: status, epic link, last activity timestamp
- Auth: OAuth token via `TokenSource`
- Interval: configurable, default 5m

### Engine Assembly

1. Each poller writes its raw data to its SQLite table
2. Engine reads all three tables
3. Matches Jira → PRs (by regex on branch/title/body)
4. Matches Worktree → PR (by branch name)
5. Matches Worktree → Jira (by regex or manual link)
6. Leftovers → UntrackedWorktree or UntrackedPR
7. Computes derived state and attention reasons
8. Broadcasts changes via WebSocket

Partial poll failure: use succeeded sources, show error indicator for failed ones.

## TUI

### Layout

Priority-based, two sections:

```
── Needs Attention (3)
   aro-hcp.fleet-caching        [no jira] → Create Jira
   sdp-pipelines.ev2-generator  ARO-14105  PR#412 ⧖ 2 comments
   aro-hcp.phase2-fleet-api     ARO-14180  PR#5920 ✗ CI failing
── On Track (2)
   aro-hcp.diagram              ARO-14201  PR#5925 ✓
   ARO-Tools.fleet-caching      ARO-14105  PR#413  ⧖ review pending
```

### Attention Triggers

- Worktree with no Jira linked
- PR with no Jira linked
- PR with failing CI checks
- PR with unresolved review comments (user hasn't replied)
- PR with merge conflicts
- PR approved but not merged
- PR with no reviewers assigned
- Jira ticket stale (no activity beyond `stale_threshold`)
- Jira status desynced from PR state (e.g. "Code Review" but PR merged)
- PR merged but worktree still exists
- PR merged but Jira still open

### Interaction

- Cursor navigates every row
- `enter` → context-sensitive action menu
- `o` → open upstream in browser
- `R` → force refresh all pollers
- `q` → quit
- `?` → help

### Action Handling

- **Simple actions** (y/n confirmation): inline in TUI. Example: "Rebase this branch?"
- **Complex input** (text editing): suspend TUI, open `$EDITOR` with pre-filled content. Example: edit Jira description before creation.

### Startup

1. Render empty dashboard
2. Run git poller (local, instant) → show worktrees
3. Run GitHub + Jira pollers async → backfill, show updates as they arrive

## AI Features (v1)

### Jira Ticket Drafting

Triggered when user selects "Create Jira" on an untracked worktree/PR.

**Context gathering** (best-effort, degrade gracefully):
1. Branch name
2. Commit messages (if any)
3. Changed files / packages
4. Claude session transcript from worktree (if `.claude/` directory exists)

**Flow:**
1. AI gathers context from available sources
2. Searches existing Jira tickets for semantic matches
3. Presents options:
   - "Link to ARO-12345: Fleet API caching improvements" (existing match)
   - "Link to ARO-12390: Performance work for HCP" (weaker match)
   - "Create new ticket" → pre-filled draft
4. On "Create new": opens `$EDITOR` with AI-drafted summary + description
5. User edits, saves, quits → Pulse creates ticket via Jira API

**Ticket defaults from config:**
- Project: ARO
- Component: aro-hcp-1p
- Issue type: Story
- Assignee: authenticated user

### Epic Suggestion

When creating a new Jira ticket, AI searches active epics under the configured component. Proposes the best semantic match based on the drafted ticket content. User confirms or overrides.

### PR Description Enrichment

When a worktree is linked to a Jira ticket and a PR exists, Pulse ensures the Jira key appears in the PR description. Best-effort, non-blocking.

## Out of Scope (v1)

- Azure DevOps integration
- Prow integration beyond GitHub checks
- Review comment AI summarization
- Copilot review validation
- Automated rebases
- Epic visibility in TUI
- launchd / auto-start
- iOS client
- Multi-user / team features
- Git hooks (pre-push Jira enforcement)
- Severity levels on attention items

## Dependencies

| Dependency | Purpose |
|------------|---------|
| `charmbracelet/bubbletea` | TUI framework |
| `spf13/cobra` | CLI framework |
| `sigs.k8s.io/yaml` | YAML parsing (strict mode) |
| `google/go-github` | GitHub API client |
| `ctreminiom/go-atlassian` | Jira API client |
| `anthropics/anthropic-sdk-go` | LLM via Vertex AI |
| `modernc.org/sqlite` | Pure-Go SQLite (no CGO) |
| `gorilla/websocket` | WebSocket server |
| `go-logr/logr` | Structured logging |

## Technical Conventions

- `sigs.k8s.io/yaml` with `UnmarshalStrict`, `json:` struct tags
- `len(x) == 0` for empty checks
- `logr` for all logging, never stdlib `log`
- Cobra CLI with RawOptions → ValidatedOptions → CompletedOptions pattern
- Time config fields as Go duration strings
- XDG-compliant paths: config in `~/.config/pulse/`, state in `~/.local/state/pulse/`
- Atomic file writes (tmp + rename) for token and state persistence
- Golden fixture tests for engine assembly and data transforms
- Tabular tests for Go
