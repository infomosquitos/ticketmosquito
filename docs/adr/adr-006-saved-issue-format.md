# ADR-006: Saved issue format — fields JSON + Markdown

- **Status:** Accepted
- **Date:** 2026-10-18
- **Related:** SpaceMosquito ADR-015 (pages); [ARCHITECTURE](../ARCHITECTURE.md)

## Context

Crawled Jira issues need durable on-disk artifacts for debugging, offline reading, attachment storage, and a single Markdown string for FTS / MCP.

## Decision

Store each issue under `saved/{PROJECT_KEY}/{ISSUE_KEY}/`:

| Artifact | Role |
|----------|------|
| `fields.json` | Raw Jira `fields` object (always persist) |
| `content.md` | Description + comments as Markdown (`body_format` recorded) |
| `metadata.json` | Key, jira_id, summary, paths, attachment list + `local_path` |
| `assets/attachments/` | Downloaded binaries (`{id}_{filename}`) |
| `assets/images/` | Reserved for inline media (optional later) |

**Database:** `issues.content` mirrors `content.md` for FTS. FTS is maintained by triggers on `summary` / `content`.

## Alternatives considered

| Option | Why not |
|--------|---------|
| HTML-only like early Confluence | Agents and FTS want Markdown |
| Markdown without raw fields | Loses custom fields for later promotion |
| Separate comment FTS entities | Epic OQ3 chose concat into issue content |

## Consequences

- Crawl writer and attachment downloader own this layout.
- `reindex` rebuilds FTS from DB rows (content regeneration from ADF is a future enhancement if needed).
