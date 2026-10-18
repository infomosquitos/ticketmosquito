# Search, HTTP get, and Jira MCP tools

- **Task ID:** `search-and-mcp-jira`
- **Status:** done
- **Parent / blocked by:** parent: `ticketmosquito-mvp-epic`; blocked_by: `jira-issue-crawl`

## Problem

A local mirror is useless to agents without FTS search and MCP/HTTP read APIs over the catalog.

## Goal

CLI + HTTP search (AND multi-word, summary boost) and get-by-key/id; MCP Streamable HTTP at `/mcp` with `jira_*` tools over the local catalog (not live Jira).

## Open questions

| # | Topic | Status | Decision |
|---|--------|--------|----------|
| 1 | Transport | locked | MCP Streamable HTTP only (SpaceMosquito ADR-017 pattern), same process as `serve` on `:11043` |
| 2 | Tool set (MVP) | locked | At least: `jira_search`, `jira_get_issue`, `jira_list_projects` (names may tweak; mirror confluence_* roles) |
| 3 | FTS fields | locked | Summary + issue `content` (description+comments MD); promote key/id/project/updated for filters |

## Non-goals

- Live Jira proxy tools
- Vector/semantic search
- Starting crawls via MCP (CLI/REST/extension only, like SpaceMosquito)

## Implementation

`migrations/sqlite/002_issues_fts`; `internal/store/sqlite/search.go`; `internal/search` DTOs; `internal/mcp` Streamable HTTP; catalog HTTP + CLI `search` / `get-issue` / `reindex`. Docs: `docs/guides/configure-mcp.md`.

## Done when

- [x] `ticketmosquito search "…"` returns ranked hits from crawled data.
- [x] `get-issue` / HTTP equivalent returns Markdown content + key metadata.
- [x] MCP client can call search + get against `http://127.0.0.1:11043/mcp`.
- [x] Configure-MCP style note in docs (can be finalized in docs task).

## Notes

SpaceMosquito: `internal/mcp`, `internal/search`, FTS5 BM25. Tools renamed from `confluence_*`.
