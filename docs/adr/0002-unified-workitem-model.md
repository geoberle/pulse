# Unified WorkItem model with K8s-style TypeMeta/Spec

Pulse represents five kinds of entities (Jira issues, PRs, CI checks, review comments, local worktrees) in a single recursive tree. We use a unified WorkItem struct with common metadata fields (Kind, ID, Label, Status) and a type-specific `Spec` field stored as `json.RawMessage`, dispatched to typed Go structs on unmarshal based on Kind. This mirrors the Kubernetes API object pattern of TypeMeta + ObjectMeta + Spec.

## Considered Options

**Per-type structs** — `JiraItem`, `PRItem`, `CheckItem`, etc. Each with its own fields. Type-safe at compile time, no casting. Rejected because the tree is recursive (`Children []*WorkItem`) and rendering walks it generically — per-type structs force the tree into `[]any` or an interface, losing the simplicity. The informer's diff logic would also need per-type branches.

**Flat struct with all fields** — one struct, every possible field present, unused ones zero-valued. Simple, no unmarshal dispatch. Rejected because it grows unbounded as kinds are added, field names collide across types (e.g. `Status` means different things for a Jira issue vs a CI check), and it's unclear which fields apply to which kind without reading the code.

**TypeMeta + Spec as json.RawMessage** (chosen) — common fields are always available for rendering and diffing. Type-specific data lives in Spec, unmarshaled on demand into typed structs (JiraSpec, PRSpec, ReviewSpec, CheckSpec, LocalSpec). Adding a new kind means adding a new Spec struct and a case in the unmarshal switch — the tree, renderer, informer, and persistence layer don't change. `Kind` is a typed `string` enum (not a bare string) with a `Validate()` method, matching the pattern used for all domain enums (`StalenessState`, `BranchState`).

## Consequences

- Serialization format is stable: state.json contains the full tree with Spec as raw JSON. New kinds are forward-compatible — old versions skip unknown Specs gracefully (ParsedSpec remains nil).
- Type-specific logic requires a type switch or assertion on ParsedSpec. This is a small cost paid in action menu logic and prompt template rendering, not in the hot path (tree walking, diffing, rendering).
- Every Spec type validates its required fields during unmarshal. Missing required fields (e.g. `JiraSpec.Key`, `PRSpec.Repo`) produce an error at parse time, not at use time.
- IDs follow `{source}:{identifier}` pattern, making them self-documenting and collision-free across kinds. The `{identifier}` part is opaque — its format is owned by the source (e.g. Supacode percent-encodes filesystem paths). Never parse, decode, or construct identifiers; treat them as passthrough values.

## Addendum: Full K8s object structure

The original design used a custom `ObjectMeta` (ID, Label, Status, ParentID) alongside TypeMeta + Spec. After adopting `cache.SharedIndexInformer`, we aligned WorkItem fully with the K8s resource shape:

- **`metav1.ObjectMeta`** replaces the custom ObjectMeta. `Name` holds the domain ID. Display text lives in kind-specific Spec fields (`JiraSpec.Summary`, `PRSpec.Title`, etc.), accessed via `DisplayName()` method. `OwnerReferences` expresses the parent-child relationship (replacing the `ParentID` string field). WorkItems are non-namespaced — the informer cache key is just `Name`.
- **`WorkItemStatus`** with `Phase` + `Conditions` replaces the inline `Status` string. Phase is the primary display status (kind-specific: `"InProgress"`, `"Open"`, `"Failed"`). Conditions are orthogonal boolean signals (`Stale`, `BranchOutdated`) using `metav1.Condition`, updated via `meta.SetStatusCondition`. This follows the Pod pattern (Phase for lifecycle, Conditions for orthogonal health signals).
- **OwnerReferences for parent-child**: `APIVersion` = `"pulse.dev/v1"`, `Kind` = parent's Kind, `Name` = parent's ID, `UID` = `""`. Set by `Flatten`, cleared by `BuildTree`. Replaces the `ParentID` string field. The semantic fit is exact — OwnerReferences express "this object belongs to that object," which is the parent relationship.

The trade-off: OwnerReferences carries two unused fields (`UID`, `BlockOwnerDeletion`) per parent relationship, and `metav1.ObjectMeta` has ~30 fields we don't use. The cost is minor JSON noise on unused fields (most are omitempty). The benefit is a WorkItem that is a standard K8s resource shape — TypeMeta, ObjectMeta, Spec, Status — with no custom one-off fields.
