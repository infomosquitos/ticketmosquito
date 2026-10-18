# ADR-002: SQLite-only distribution

- **Status:** Accepted
- **Date:** 2026-10-18
- **Related:** SpaceMosquito ADR-016

## Context

TicketMosquito is a local-first Jira mirror. A separate database service adds install friction without MVP benefit.

## Decision

Use **SQLite + WAL** only (`modernc.org/sqlite`). All state lives under `~/.ticketmosquito` (or `--data-dir` / `TICKETMOSQUITO_DATA_DIR`). Schema via `golang-migrate` with embedded SQL for release builds.

## Consequences

- Single-file DB (`ticketmosquito.db`) next to `config.yaml`, `session.enc`, `saved/`.
- FTS5 for search lands in a later task; initial schema has `projects` + `issues` only.
