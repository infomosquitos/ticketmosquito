# Spike: Jira API & content model

- **Task ID:** `spike-jira-api-content-model`
- **Status:** done
- **Parent / blocked by:** parent: `ticketmosquito-mvp-epic`

## Problem

TicketMosquito’s crawl pipeline depends on how Jira exposes issues, bodies, comments, attachments, and fields — and how that differs between Cloud and Server/DC. Guessing wrong wastes a full scaffold rewrite.

## Goal

Produce a short written recommendation (in this task’s Notes, or a small `docs/` draft) that locks enough API/content facts for the MVP epic’s open questions on crawl unit, fields, comments, attachments, and body conversion.

## Open questions

| # | Topic | Status | Decision |
|---|--------|--------|----------|
| 1 | Primary discovery API (JQL search vs project endpoints) and pagination model for Cloud and Server/DC | locked | **JQL** `project = KEY` (matches epic: project-key crawl unit). **Cloud:** `GET/POST /rest/api/3/search/jql` with `nextPageToken` (legacy `/search` removed). **Server/DC:** `GET/POST /rest/api/2/search` with `startAt`/`maxResults`/`total`. Optional Cloud optimize: IDs-only search then `POST /rest/api/3/issue/bulkfetch`. |
| 2 | Canonical issue fetch expand/fields for description, comments, attachments, changelog (in/out of MVP) | locked | Per issue: `GET …/issue/{keyOrId}?fields=*all` (or explicit list) + `expand=renderedFields`. Comments: dedicated `GET …/issue/{id}/comment?expand=renderedBody` with pagination (search embeds cap ~20 comments on Cloud enhanced search — do **not** rely on search for full comment history). Attachments: from issue `fields.attachment` metadata + content download URLs. **Changelog out of MVP.** |
| 3 | Description/comment body representation(s): ADF, wiki markup, rendered HTML — which for Cloud vs DC | locked | **Cloud:** description/comment `body` = **ADF**; HTML via `renderedFields.description` / comment `renderedBody`. **Server/DC:** description/comment body often **wiki markup string**; HTML via `expand=renderedFields`. Primary path: structured body → Markdown (ADF→MD Cloud; wiki→MD or HTML→MD on DC). Fallback: rendered HTML → generic HTML→MD (epic OQ5). |
| 4 | Attachment download URLs and auth (cookie session sufficient?) | locked | Use attachment `content` URL from metadata, or Cloud `GET /rest/api/3/attachment/content/{id}`. Send session **Cookie** header + **`X-Atlassian-Token: no-check`** (same as SpaceMosquito). Cookie session is the MVP auth path; API token/basic is optional later. Expect HTML login page if session expired (detect like SM asset downloader). |
| 5 | Stable identity: issue `id` vs `key`; how updates are detected (updated timestamp, fields hash, etc.) | locked | Store both: numeric **`id`** (stable) and **`key`** (human; can change on move — rare). Incremental: compare remote **`fields.updated`** to local (epic OQ1). `--force` ignores skip. |
| 6 | Minimum field list for MVP search/MCP usefulness | locked | Align epic OQ2: always persist **raw fields JSON**; promote **`key`**, **`id`**, **`summary`**, **`description`**, plus **`updated`** for incremental; FTS = summary + description MD + comments MD (OQ3 A). Also fetch `attachment` (+ comment bodies) for crawl completeness even if not promoted columns. |

## Non-goals

- Implementing the crawler or converters
- Exhaustive coverage of every Jira custom field type
- Deciding SpaceMosquito code-reuse strategy (see sibling spike)

## Implementation

Analysis only — completed 2026-10-18 from Atlassian Cloud REST v3 docs, Server/DC REST v2 docs, and enhanced-search migration notes (legacy Cloud `/search` removed ~2025).

## Done when

- [x] Notes section (or linked draft doc) answers the open questions with proposed `locked` candidates.
- [x] Explicit Cloud vs Server-DC differences called out where they matter for MVP.
- [x] Epic `ticketmosquito-mvp-epic` open questions 1–5 can be reviewed by a human for locking.
- [x] No production crawler code required for this spike.

## Notes

### Mapping SpaceMosquito → TicketMosquito

| SpaceMosquito | TicketMosquito |
|---------------|----------------|
| Discover space pages (`/space/{key}/content/page?expand=version`) | Discover project issues via JQL `project = KEY` (+ `ORDER BY updated ASC` optional) |
| Page version skip | `fields.updated` skip |
| `GET content/{id}?expand=body.storage,…` | `GET issue/{key}?fields=…&expand=renderedFields` + paginated comments |
| CSF → MD / HTML → MD | ADF→MD (Cloud) or wiki/HTML→MD (DC) / rendered HTML→MD fallback |
| `/download/attachments/…` + session | Attachment `content` URL or `/attachment/content/{id}` + session |
| `saved/{space}/{id}-{title}/` | `saved/{projectKey}/{issueKey}/` (recommend; finalize in impl task) |

### Flavor detection

Same idea as SpaceMosquito session flavor:

- Host `*.atlassian.net` → try Cloud (`/rest/api/3/myself`)
- Else try Server/DC (`/rest/api/2/myself`), honor context path if present
- Store flavor on session; build API base from it

### Recommended crawl pipeline (MVP)

```
project KEY
  → search JQL "project = KEY" (Cloud: /search/jql + nextPageToken; DC: /search + startAt)
      fields for discovery: id, key, updated  (lightweight)
  → for each issue:
      if !force && local.updated >= remote.updated → skip
      GET issue (fields=*all or broad list, expand=renderedFields)
      paginate GET …/comment?expand=renderedBody
      write raw fields JSON + convert description/comments → content.md
      download attachments into assets/
      upsert SQLite + FTS (summary + content.md)
```

**Cloud large projects:** optional later — search with no/minimal fields to collect IDs (maxResults up to ~5000), then parallel `POST /rest/api/3/issue/bulkfetch` for details. Not required for first crawl slice.

### Body conversion detail

| | Cloud | Server/DC |
|--|-------|-----------|
| Primary body | ADF JSON on `fields.description` / comment `body` | Wiki markup string (typical) |
| Rendered HTML | `expand=renderedFields` → `renderedFields.description`; comments `expand=renderedBody` | `expand=renderedFields` |
| MD strategy | ADF→Markdown library/rules (CSF analogue); else HTML→MD | Prefer rendered HTML→MD if wiki converter is weak; or wiki→MD if simple |

Store on disk (SpaceMosquito-like): raw structured body + rendered HTML when available + `content.md` + `body_format` in metadata (`adf` \| `wiki` \| `rendered`).

### Comments

- Full history requires **comment API pagination** (`startAt`/`maxResults`; Cloud default page size often 100).
- Do not trust comments embedded in search/issue expand alone for complete history (Cloud enhanced search limits embedded comments).
- Indexing: concatenate into issue Markdown (epic OQ3 A).

### Attachments

- Metadata: `fields.attachment[]` (`id`, `filename`, `mimeType`, `content`, `size`, …).
- Download: follow `content` URL with cookies; Cloud also documents `GET /rest/api/3/attachment/content/{id}`.
- Same skip/`--force` semantics as SpaceMosquito asset downloader.
- Private/restricted comments may hide some attachments — respect API omissions.

### Auth & risks

| Risk | Note |
|------|------|
| Session cookies | Works for browser SSO like SpaceMosquito; Cloud may prefer API tokens for non-interactive — MVP cookie-first |
| Rate limits | Cloud has rate limiting; backoff/retry on 429 |
| Enhanced search consistency | Cloud `/search/jql` may lag recent writes; fine for mirror; `reconcileIssues` exists if needed later |
| Permissions | Browse project + issue security + comment visibility restrict what the session user sees |
| ADF complexity | Tables, mentions, panels — “good enough” MD like CSF; HTML fallback for hard nodes |
| Issue key changes | Project move renames key; keep numeric `id` as durable PK |
| Context path (DC) | e.g. `/jira` — detect like Confluence context path |

### Epic alignment

Epic OQ1–5 already locked; this spike **confirms** they are API-feasible and supplies concrete endpoints for implementation tasks. No epic OQ changes required.

### Sources (consulted)

- [Jira Cloud REST — Issue search](https://developer.atlassian.com/cloud/jira/platform/rest/v3/api-group-issue-search/) (`/search/jql`, `nextPageToken`)
- Atlassian KB: legacy `/rest/api/3/search` removed; migrate to `/search/jql`
- Enhanced search migration / bulkfetch guidance (RFC-61 / community posts)
- [Jira Cloud — Issue attachments](https://developer.atlassian.com/cloud/jira/platform/rest/v3/api-group-issue-attachments/)
- [Jira Server REST examples](https://developer.atlassian.com/server/jira/platform/jira-rest-api-examples/) (`/rest/api/2/search`, `startAt`)
- Server expand=`renderedFields` tutorial; Cloud `renderedBody` for comments
