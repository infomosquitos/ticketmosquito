# Spike: reuse boundary from SpaceMosquito

- **Task ID:** `spike-reuse-from-spacemosquito`
- **Status:** done
- **Parent / blocked by:** parent: `ticketmosquito-mvp-epic`

## Problem

TicketMosquito should reuse SpaceMosquito’s **idea and technology**, but a blind copy of the Confluence-shaped codebase will fight issue/comment/attachment semantics. We need a clear reuse boundary before scaffolding.

## Goal

Document what to reuse as patterns (or paste-adapt), what to redesign, and a recommended relationship between the two repos (independent greenfield vs selective copy vs future shared module) — enough to lock epic questions 6–8.

## Open questions

| # | Topic | Status | Decision |
|---|--------|--------|----------|
| 1 | Repo strategy: independent greenfield mirroring patterns vs large paste from SpaceMosquito packages | locked | Greenfield + selective paste; no published shared module for MVP |
| 2 | Which packages are “shell” (paths, session crypto, sqlite/FTS, MCP transport, rod job plumbing) vs domain-specific | locked | See Notes reuse table; shell = paste kernels, domain = rewrite |
| 3 | Extension: fork SpaceMosquito extensions with Jira host detection vs shared Atlassian extension | locked | New separate Ticket Mosquito extension (epic OQ8) |
| 4 | Port / data-dir / binary naming to avoid colliding with a running SpaceMosquito | locked | `ticketmosquito`, port `11043`, `~/.ticketmosquito` / `TICKETMOSQUITO_DATA_DIR` |
| 5 | Which SpaceMosquito ADRs apply unchanged vs need TicketMosquito ADRs | open | Copy ADR *topics* as TicketMosquito ADRs when implementing; not a shared package |

## Non-goals

- Actually extracting a shared Go module in this spike
- Implementing TicketMosquito packages
- Deep Jira API research (sibling spike)

## Implementation

Analysis only, using `/Users/ruslanbes/projects/SpaceMosquito`:

1. Classify `internal/*` (and extension) as reusable shell / adapt / replace.
2. Propose first implementation slice after spikes (scaffold order).
3. Recommend naming (binary, env vars, default port, data dir).
4. List ADRs to copy vs rewrite for TicketMosquito.

## Done when

- [ ] Reuse table (shell / adapt / replace) written in Notes.
- [ ] Recommended repo strategy and naming defaults proposed for epic questions 6–8.
- [ ] Suggested first 3–5 implementation tasks listed (still not created unless asked).
- [ ] No requirement to change SpaceMosquito.

## Notes

### Verdict (2026-10-18)

**Almost nothing is worth publishing as a shared library for two products.** Reusable value is design DNA + a few small kernels. Domain code (store, scraper, CSF, MCP tools, API) is Confluence-shaped. Prefer **greenfield + selective paste**; extract/publish only if a third sibling appears or you are double-fixing the same crypto/transport bugs.

### What could be extracted (theoretical)

| Tier | Piece | ~Size | Publish now? |
|------|-------|-------|--------------|
| A | AES-GCM multi-host session blob (`session/store.go`) | ~360 LOC | No — paste; rename `ConfluenceURL` → site URL; rewrite Jira validation probes |
| A | Data-dir / paths pattern | ~370 LOC | No — copy with `TICKETMOSQUITO_*` / `~/.ticketmosquito` |
| A | Generic HTML→Markdown (`contentmd/markdown.go` only) | ~110 LOC | No — paste + new selectors |
| A | Asset downloader (`storage/asset.go`) | ~390 LOC | No — selective paste |
| A | Rod Chromium launcher | ~100 LOC | No — paste if browser fallback stays |
| B | MCP Streamable HTTP transport shell | ~100 LOC glue | No — tools/DTOs are product; framework tax |
| B | Crawl job status machine | mixed | No — runner is domain |
| C | CSF converter, store/schema, scraper, API, `confluence_*` MCP, cron spaces | large | Rewrite for issues |

### Proposed shared modules (only if extracting later)

- `mosquito-session` — encrypted hostname→session map; product supplies validation
- `mosquito-paths` — parameterized product name / env / default dir
- `htmlmd` / `assetfetch` — optional thin helpers

Not justified for TicketMosquito MVP alone.

### Recommended OQ6 answer

**Independent greenfield repo; selective paste of patterns from SpaceMosquito; do not publish a shared Go module now.** Rewrite issue domain (schema, crawl, ADF/HTML, MCP tools, extension hosts). Revisit extraction when a third product shares the shell.
