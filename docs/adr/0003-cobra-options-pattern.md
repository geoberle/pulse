# Cobra with RawOptions/ValidatedOptions/CompletedOptions

Pulse uses cobra for CLI bootstrapping with the three-stage options pattern from ARO-HCP templatize: RawOptions (flag bindings) → ValidatedOptions (parsed config, checked invariants) → CompletedOptions (live clients, opened resources). This separates flag parsing, validation, and resource acquisition into distinct phases enforced by the type system.

## Considered Options

**Simple flag parsing in RunE** — parse flags, create clients, run. Works for a small CLI. Rejected because Pulse's startup has meaningful validation (Jira auth, GitHub token extraction, config file parsing) and expensive resource acquisition (API clients, LLM client, persisted state loading) that benefit from explicit phases. Mixing these in one function makes error handling and testing harder.

**Viper + struct tags** — automatic config binding. Rejected because it hides the validation logic behind reflection and doesn't enforce phase ordering. A misconfigured Jira host surfaces at API call time instead of startup.

**Three-stage options** (chosen) — each stage is a separate type. You can't call `Complete()` without first calling `Validate()` — the compiler enforces it. Validate checks config invariants (required fields, valid intervals). Complete opens real resources (HTTP clients, socket connections). RunE is three lines: validate, complete, run. Each stage is independently testable.

## Consequences

- More boilerplate than a simple RunE, but startup failures are caught early with clear error messages.
- Future subcommands (`pulse config`, `pulse status`) compose by embedding parent options, chaining validation.
- The pattern is familiar within the ARO-HCP codebase, reducing context-switching cost.
