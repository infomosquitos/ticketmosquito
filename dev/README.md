# Development

How we run this project: ground rules, setup, cleanup, release, and current work tracking.

## Local setup

**Prerequisites:** Go 1.25+ (tested with 1.26).

```sh
# Build
make build
# or: go build -o ticketmosquito ./cmd/ticketmosquito

# First-time data dir + migrations (portable example)
./ticketmosquito init --data-dir ./data
TICKETMOSQUITO_DATA_DIR=./data ./ticketmosquito serve
# listens on http://127.0.0.1:11043  — GET /healthz ; session at /api/session*

# Default data dir is ~/.ticketmosquito
./ticketmosquito init
./ticketmosquito serve

# Browser extensions (session capture)
make build-extensions
# then load chrome-extension/dist or firefox-extension/dist — see docs/guides/extension-install.md

# Tests
make test

# Search / get (after crawl)
./ticketmosquito search "login bug"
./ticketmosquito get-issue PROJ-1
```

Module path: `github.com/infomosquitos/ticketmosquito`. Default API/MCP port: **11043**. MCP: see [`docs/guides/configure-mcp.md`](../docs/guides/configure-mcp.md).

## Ground rules

- Track work here in `dev/` — flat backlog, status, optional task detail files under `tasks/`.
- Work one task at a time.
- Before implementing, check for open questions; lock them in the task doc (or an ADR) instead of guessing.
- Use [Mermaid](https://mermaid.js.org/) diagrams for architecture documentation.
- When cleaning finished or cancelled work, follow [`CLEANUP.md`](CLEANUP.md). When cutting a version, follow [`RELEASE.md`](RELEASE.md).
- Do not choose a tech stack until recorded in an ADR / this README (Go + SQLite already decided — see ADRs 001–002).
- Prefer selective paste from SpaceMosquito patterns; no shared published library for MVP.

## What's in this folder

| Doc | Role | Lifetime |
|-----|------|----------|
| [`README.md`](README.md) (this file) | Setup, ground rules, doc index | Durable |
| [`CLEANUP.md`](CLEANUP.md) | Promote durable content; remove finished tasks | Durable |
| [`RELEASE.md`](RELEASE.md) | Version changelog, bump version sources, tag | Durable |
| [`BACKLOG.md`](BACKLOG.md) | Flat task index | Living — cleaned via CLEANUP.md |
| [`STATUS.md`](STATUS.md) | Current focus / blockers / session notes | Living — refreshed each session |
| [`tasks/`](tasks/) | Optional detail for a backlog item | Temporary — deleted when the task ships or is cancelled |

## Reference docs

| Doc | Role |
|-----|------|
| [`README.md`](../README.md) | User-facing get-started |
| [`docs/ARCHITECTURE.md`](../docs/ARCHITECTURE.md) | System sketch |
| [`docs/INSTALL-FROM-SOURCE.md`](../docs/INSTALL-FROM-SOURCE.md) | Build from clone |
| [`docs/adr/`](../docs/adr/) | ADRs 001–006 (Go, SQLite, session, YAML, MCP, saved format) |
| [`docs/guides/configure-mcp.md`](../docs/guides/configure-mcp.md) | MCP Streamable HTTP |
| [`docs/guides/extension-install.md`](../docs/guides/extension-install.md) | Load Chrome/Firefox extension |
| [`CHANGELOG.md`](../CHANGELOG.md) | Shipped user-facing history |
| SpaceMosquito (reference) | patterns only — selective paste |
