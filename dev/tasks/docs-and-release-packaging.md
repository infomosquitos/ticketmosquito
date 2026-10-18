# Docs, ADRs, and release packaging

- **Task ID:** `docs-and-release-packaging`
- **Status:** done
- **Parent / blocked by:** parent: `ticketmosquito-mvp-epic`; blocked_by: `search-and-mcp-jira`, `jira-attachments-storage`, `jira-session-and-extension`

## Problem

MVP code without README/architecture/ADRs and a release shape won’t be installable or maintainable the SpaceMosquito way.

## Goal

User-facing README (init → session → crawl → search/MCP), architecture sketch, key ADRs, and a GitHub release workflow for the binary + extension artifacts (parity with SpaceMosquito install story, TicketMosquito naming/ports).

## Open questions

| # | Topic | Status | Decision |
|---|--------|--------|----------|
| 1 | ADR set (MVP) | locked | At least: Go backend, SQLite-only, YAML config, hybrid session, MCP Streamable HTTP, saved-issue format — adapted from SpaceMosquito ADR topics |
| 2 | Release artifacts | locked | Single Go binary + Firefox XPI / Chrome zip from CI when ready; document from-source build first if CI lands last |

## Non-goals

- Marketing site
- Docker/Postgres distribution

## Implementation

`README.md`, `docs/ARCHITECTURE.md`, `docs/INSTALL*.md`, ADRs 004–006, MCP/extension guides, `.github/workflows` (test + tag release), `scripts/build-*.sh`, CHANGELOG Unreleased for MVP.

## Done when

- [x] New user can follow README from zero to crawl + MCP on port `11043`.
- [x] Architecture doc matches implemented components.
- [x] ADRs accepted for stack/auth/storage/MCP decisions.
- [x] Release or from-source install path documented; CHANGELOG has Unreleased notes for shipped MVP behavior.

## Notes

Mirror SpaceMosquito docs structure lightly; Jira naming throughout. Firefox AMO-signed XPI optional later; CI ships unsigned firefox zip + chrome zip.
