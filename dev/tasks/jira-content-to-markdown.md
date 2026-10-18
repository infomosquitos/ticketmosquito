# Description and comments → Markdown

- **Task ID:** `jira-content-to-markdown`
- **Status:** done
- **Parent / blocked by:** parent: `ticketmosquito-mvp-epic`

## Problem

Issue description and comments arrive as ADF (Cloud), wiki markup (DC), or rendered HTML. Search/MCP need a single Markdown `content` string (epic OQ3/OQ5).

## Goal

A converter package that turns issue description + comments into searchable Markdown, with structured-body primary and rendered-HTML fallback, and records `body_format` for reindex later.

## Open questions

| # | Topic | Status | Decision |
|---|--------|--------|----------|
| 1 | Dual path | locked | Structured → MD primary; rendered HTML → MD fallback (epic OQ5) |
| 2 | Comments in FTS | locked | Append comments into issue Markdown (epic OQ3 A) |
| 3 | ADF library vs hand rules | locked | **Prefer an existing Go ADF→Markdown library.** Use rendered HTML→MD as fallback when conversion fails or is empty (epic OQ5). Hand-rolled ADF rules only if no viable library. → `github.com/rgonek/jira-adf-converter` |

## Non-goals

- Perfect ADF fidelity (panels, all macros)
- Separate comment FTS entities
- Live reindex CLI (can stub; full `reindex --content` may land with search task)

## Implementation

`internal/contentmd`: ADF→MD, wiki heuristics, HTML→MD. `RenderIssue` concatenates comments under `## Comments`.

## Done when

- [x] Cloud-like ADF description+comments → usable Markdown in tests.
- [x] Rendered HTML path produces Markdown when structured conversion fails/empty.
- [x] Comment blocks are clearly delimited in the MD (author/date headings OK).
- [x] `body_format` convention documented for storage writer (`internal/contentmd/README.md`).

## Notes

Library: `github.com/rgonek/jira-adf-converter`. HTML: SpaceMosquito-style `html-to-markdown/v2`.
