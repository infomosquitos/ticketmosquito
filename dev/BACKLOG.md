# Backlog

Flat task list. Status values: `backlog` | `ready` | `in_progress` | `blocked` | `done` | `cancelled`.

On cleanup, move shipped work to [`CHANGELOG.md`](../CHANGELOG.md) and clean the backlog — see [`CLEANUP.md`](CLEANUP.md).

Reorder freely. Cancel by setting status to `cancelled`. Unexpected work becomes a new task with its own ID.

## Format

Each task:

```markdown
## <task-id>
- status: backlog
- parent: <optional-related-task-id>
- blocked_by: <optional-task-id>
- goal: One sentence
- done_when: Observable acceptance criteria
- notes: Optional one-liner
```

Detail lives in `dev/tasks/<task-id>.md` when needed.

---

## ticketmosquito-mvp-epic
- status: done
- goal: Local Jira mirror (crawl issues → SQLite/FTS + MCP) reusing SpaceMosquito’s product idea and tech shape.
- done_when: See `dev/tasks/ticketmosquito-mvp-epic.md`.
- notes: MVP implementation + docs/release packaging complete.

## spike-jira-api-content-model
- status: done
- parent: ticketmosquito-mvp-epic
- goal: Recommend Jira REST/content model for discovery, bodies, comments, attachments, fields (Cloud + DC).
- done_when: See `dev/tasks/spike-jira-api-content-model.md`.
- notes: Cloud /search/jql + nextPageToken; DC /search + startAt; ADF vs wiki; comments paginated.

## spike-reuse-from-spacemosquito
- status: done
- parent: ticketmosquito-mvp-epic
- goal: Define reuse boundary and naming vs SpaceMosquito before scaffolding.
- done_when: See `dev/tasks/spike-reuse-from-spacemosquito.md`.
- notes: Greenfield + selective paste; no shared lib.

## scaffold-go-sqlite
- status: done
- parent: ticketmosquito-mvp-epic
- goal: Greenfield Go binary with init/serve, data dir, YAML, SQLite schema for projects/issues.
- done_when: See `dev/tasks/scaffold-go-sqlite.md`.
- notes: Module github.com/infomosquitos/ticketmosquito; port 11043; projects+issues schema.

## jira-session-and-extension
- status: done
- parent: ticketmosquito-mvp-epic
- goal: Encrypted Jira cookie session + new Ticket Mosquito browser extension.
- done_when: See `dev/tasks/jira-session-and-extension.md`.
- notes: POST/DELETE/status/validate /api/session; Chrome+Firefox extensions; ADR-003.

## jira-content-to-markdown
- status: done
- parent: ticketmosquito-mvp-epic
- goal: ADF/wiki/HTML → Markdown for description + comments (FTS concat).
- done_when: See `dev/tasks/jira-content-to-markdown.md`.
- notes: rgonek/jira-adf-converter + html-to-markdown; body_format adf|wiki|rendered.

## jira-issue-crawl
- status: done
- parent: ticketmosquito-mvp-epic
- goal: Crawl project by key with updated-skip / --force; save raw fields + content.md + DB upsert.
- done_when: See `dev/tasks/jira-issue-crawl.md`.
- notes: API-only; Cloud search/jql + DC search; httptest fixtures for incremental/--force.

## jira-attachments-storage
- status: done
- parent: ticketmosquito-mvp-epic
- blocked_by: jira-issue-crawl
- goal: Download attachments into per-issue assets/ like SpaceMosquito.
- done_when: See `dev/tasks/jira-attachments-storage.md`.
- notes: assets/attachments/{id}_{filename}; skip unless --force; best-effort failures.

## search-and-mcp-jira
- status: done
- parent: ticketmosquito-mvp-epic
- blocked_by: jira-issue-crawl
- goal: FTS search, get-issue, MCP jira_* tools on local catalog.
- done_when: See `dev/tasks/search-and-mcp-jira.md`.
- notes: FTS5 BM25; CLI/HTTP search+get; MCP /mcp with jira_search|get_issue|list_projects.

## docs-and-release-packaging
- status: done
- parent: ticketmosquito-mvp-epic
- blocked_by: search-and-mcp-jira, jira-attachments-storage, jira-session-and-extension
- goal: README, architecture, ADRs, install/release packaging for MVP.
- done_when: See `dev/tasks/docs-and-release-packaging.md`.
- notes: README + ARCHITECTURE + INSTALL; ADRs 004–006; CI test/release; build-release.sh.

## homebrew-tap-distribution
- status: backlog
- blocked_by: spike-homebrew-tap-formula, spike-extension-coinstall
- goal: Custom Homebrew tap so macOS users get the binary plus a clear extension co-install path.
- done_when: See `dev/tasks/homebrew-tap-distribution.md`.
- notes: Brew cannot silent-install browser extensions; spikes own formula shape + assisted install UX.

## spike-homebrew-tap-formula
- status: backlog
- parent: homebrew-tap-distribution
- goal: Recommend tap layout, formula vs cask, bottle/asset mapping, platforms.
- done_when: See `dev/tasks/spike-homebrew-tap-formula.md`.

## spike-extension-coinstall
- status: backlog
- parent: homebrew-tap-distribution
- goal: Recommend how Chrome/Firefox extensions ship with the app without silent browser install.
- done_when: See `dev/tasks/spike-extension-coinstall.md`.
- notes: Architecture changes (CLI, share/, data-dir) allowed.

## upgrade-extensions-typescript-7
- status: backlog
- goal: Make chrome/firefox extension builds work on TypeScript 7 (replace ts-loader-era API assumption).
- done_when: See `dev/tasks/upgrade-extensions-typescript-7.md`.
- notes: Dependabot TS7 merge broke ts-loader; pinned ~5.9.3 for now.

## extension-three-tab-parity
- status: done
- goal: SpaceMosquito-shaped 3-tab extension (Issue/Projects/Settings) for Jira, plus full crawl/compare/cron HTTP APIs.
- done_when: See `dev/tasks/extension-three-tab-parity.md`.
- notes: Backend projects/crawl/issue ops/cron + Chrome/Firefox 3-tab UI shipped.
