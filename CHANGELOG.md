# Changelog

All notable changes to this project. Format based on [Keep a Changelog](https://keepachangelog.com/).

## [Unreleased]

## [0.1.0] - 2026-10-18

### Added

- Local Jira mirror: `init` / `serve` / crawl / search / MCP on `127.0.0.1:11043`, data dir `~/.ticketmosquito` (or `--data-dir` / `TICKETMOSQUITO_DATA_DIR`).
- Encrypted multi-host cookie session + Ticket Mosquito Chrome/Firefox extensions (Cloud and Server/DC).
- Project crawl with incremental `updated` skip, `--force`, Markdown content (ADF/wiki/HTML), comments, raw fields, attachment download.
- FTS5 search; CLI `search` / `get-issue` / `reindex`; HTTP catalog APIs; MCP Streamable HTTP (`jira_search`, `jira_get_issue`, `jira_list_projects`).
- Docs: README, architecture, install guides, ADRs; CI test and tag-release workflows.
