# Architecture Decision Records

ADRs capture **durable** choices (stack, boundaries, contracts) that should outlive any single task.

## When to write an ADR

- Choosing or changing language, frameworks, hosting, or major libraries.
- Defining system boundaries or data/API contracts that other work depends on.
- Recording a trade-off so future agents do not re-litigate it in every task.

Prefer a **task doc** (`dev/tasks/…`) for short-lived design while building a feature. On cleanup, promote lasting decisions from the task doc into an ADR before deleting the task file — see [`dev/CLEANUP.md`](../../dev/CLEANUP.md).

## Conventions

- Files: `docs/adr/adr-NNN-short-title.md` (zero-padded number, kebab-case title).
- Status: `Proposed` | `Accepted` | `Superseded` | `Deprecated`.
- Copy [`TEMPLATE.md`](TEMPLATE.md) for a new record.

## Index

| ADR | Title | Status |
|-----|-------|--------|
| [001](adr-001-go-backend-language.md) | Go backend language | Accepted |
| [002](adr-002-sqlite-only-distribution.md) | SQLite-only distribution | Accepted |
| [003](adr-003-hybrid-session-auth.md) | Hybrid browser-session auth | Accepted |
| [004](adr-004-configuration-format-yaml.md) | Configuration format — YAML | Accepted |
| [005](adr-005-mcp-streamable-http.md) | MCP Streamable HTTP | Accepted |
| [006](adr-006-saved-issue-format.md) | Saved issue format | Accepted |
