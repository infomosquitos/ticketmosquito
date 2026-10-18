# Scaffold Go binary, data dir, SQLite

- **Task ID:** `scaffold-go-sqlite`
- **Status:** done
- **Parent / blocked by:** parent: `ticketmosquito-mvp-epic`

## Problem

TicketMosquito has process docs only — no runnable binary, config, or database to hang session/crawl/MCP on.

## Goal

A greenfield Go module that builds `ticketmosquito` with `init` + `serve` stubs, YAML config, `~/.ticketmosquito` (or `--data-dir`), SQLite opened/migrated, and empty project/issue schema ready for crawl upserts.

## Open questions

| # | Topic | Status | Decision |
|---|--------|--------|----------|
| 1 | Module path / Go version | locked | `github.com/infomosquitos/ticketmosquito`; Go 1.25+ (built with 1.26) |
| 2 | Default listen address | locked | `127.0.0.1:11043` (epic OQ7) |
| 3 | Initial schema | locked | `projects` + `issues` with id/key/project, summary, content, fields_json, paths, body_format, jira_updated. FTS deferred to search task. |

## Non-goals

- Real Jira crawl, session crypto end-to-end, extension, MCP tools
- Publishing shared libraries with SpaceMosquito

## Implementation

Selective paste of SpaceMosquito patterns: `cmd/ticketmosquito`, `internal/paths`, `internal/config`, `internal/datastore`, `internal/store` (+ sqlite), migrations, `init` / `serve` / `version`. Wire empty HTTP server on `:11043`. Fill `dev/README.md` local setup.

## Done when

- [x] `go build` produces `ticketmosquito`; `init` creates data dir + `config.yaml` + DB; `serve` listens on `127.0.0.1:11043`.
- [x] Migrations create projects/issues tables (FTS deferred).
- [x] `dev/README.md` has install/run commands; ADR for Go + SQLite.
- [x] No Confluence types or SpaceMosquito import path left in the module.

## Notes

Reference: `/Users/ruslanbes/projects/SpaceMosquito/spacemosquito/` — `cmd/spacemosquito`, `internal/paths`, `internal/config`, `internal/datastore`, `migrations/sqlite`.

Shipped layout: repo-root Go module; `GET /healthz` on serve.
