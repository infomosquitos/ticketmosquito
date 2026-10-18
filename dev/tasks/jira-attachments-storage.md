# Download and store issue attachments

- **Task ID:** `jira-attachments-storage`
- **Status:** done
- **Parent / blocked by:** parent: `ticketmosquito-mvp-epic`; blocked_by: `jira-issue-crawl`

## Problem

Attachment metadata alone isn’t enough — agents and offline use need files on disk, same as SpaceMosquito page assets.

## Goal

During crawl (and `--force` refresh), download issue attachments into the per-issue `assets/` tree using session auth, skip existing files unless forced, best-effort on failure.

## Open questions

| # | Topic | Status | Decision |
|---|--------|--------|----------|
| 1 | Policy | locked | Same as SpaceMosquito (epic OQ4): no type whitelist/size cap; skip unless `--force`; cookie + `X-Atlassian-Token: no-check` |
| 2 | Layout | locked | Under `saved/{projectKey}/{issueKey}/assets/attachments/` (and `images/` if inline media URLs appear in HTML/ADF) |

## Non-goals

- OCR / indexing binary contents into FTS
- Uploading attachments back to Jira

## Implementation

`internal/storage/asset.go` (AssetDownloader + stable names); wired in scraper after issue fetch. Detect HTML-login responses as session expiry. Record `local_path` in `metadata.json`. Inline ADF/HTML image download deferred (dir helper exists).

## Done when

- [x] Attachments from a fixture/real issue land on disk with stable names.
- [x] Re-crawl without `--force` skips existing; with `--force` re-downloads.
- [x] Failed downloads log and do not abort the whole issue crawl.

## Notes

Cloud: `GET /rest/api/3/attachment/content/{id}` or attachment `content` URL. See spike § Attachments.
