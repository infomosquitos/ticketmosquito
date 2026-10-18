# Crawl Jira project issues

- **Task ID:** `jira-issue-crawl`
- **Status:** done
- **Parent / blocked by:** parent: `ticketmosquito-mvp-epic`; blocked_by: `jira-session-and-extension`, `jira-content-to-markdown`

## Problem

Without a crawler, the local catalog stays empty. Need project-key discovery, incremental fetch, raw field JSON, and Markdown content on disk + SQLite.
    10|
## Goal

`ticketmosquito crawl <project-url-or-key>` discovers all issues in a project, skips unchanged by `updated` unless `--force`, fetches full issue + paginated comments, writes `saved/` artifacts + raw fields JSON, upserts DB, and converts content via the markdown package.

## Open questions

| # | Topic | Status | Decision |
|---|--------|--------|----------|
| 1 | Crawl unit + incremental | locked | Project key; skip if local `updated` ≥ remote; `--force` refreshes all (epic OQ1) |
| 2 | Discovery APIs | locked | Cloud `/rest/api/3/search/jql` + `nextPageToken`; DC `/rest/api/2/search` + `startAt` (spike) |
| 3 | On-disk layout | locked | `saved/{projectKey}/{issueKey}/` with raw fields, content.md, metadata, assets dir |
| 4 | Browser fallback | locked | **API-only for this task.** Defer go-rod browser fallback to a follow-up if API proves insufficient. |

## Non-goals

- Saved-JQL crawl units
- Changelog
- Attachment download (see `jira-attachments-storage`) — leave hooks/metadata for attachment list OK
- MCP/search UX polish (separate task)

## Implementation

`internal/jira` client (search/get/comments) + URL helpers; `internal/scraper` crawl/skip/upsert; `internal/storage` writer; CLI `crawl` / `crawl-issue`. Persist raw `fields.json`; call contentmd; upsert `projects`/`issues`.

## Done when

- [x] Crawl a real or recorded-fixture project: issues appear under `saved/` and in SQLite with summary + content MD + raw fields.
- [x] Incremental skip works; `--force` re-fetches.
- [x] Comments paginated and included in content MD.
- [x] Cloud and DC URL construction covered by tests or dual fixtures.

## Notes

Full pipeline notes in `spike-jira-api-content-model.md`. Attachment binary download is the next task; crawl stores attachment metadata from the issue payload.
