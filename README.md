# TicketMosquito

Local **Jira** mirror: crawl issues into SQLite + FTS5, search offline, and expose an MCP server for agents. Same product shape as SpaceMosquito, but for Jira issues instead of Confluence pages.

All state lives under `~/.ticketmosquito/` (or a portable `--data-dir` / `TICKETMOSQUITO_DATA_DIR`).

Default API/MCP listen: **`127.0.0.1:11043`**.

## Install

- **[Install from source](docs/INSTALL-FROM-SOURCE.md)** — build with Go and npm (primary path today)
- **[Install](docs/INSTALL.md)** — GitHub Release binaries + extension packages (when a tagged release exists)

## First run

```sh
make build
./ticketmosquito init
./ticketmosquito serve
```

`init` writes a generated **encryption key** to `config.yaml` (`session.encryption_key`). Keep that file — the same key is required to decrypt `session.enc`.

Portable mode:

```sh
./ticketmosquito init --data-dir ./data
TICKETMOSQUITO_DATA_DIR=./data ./ticketmosquito serve
```

## Capture a Jira session

1. Start the server: `./ticketmosquito serve`
2. Load the Ticket Mosquito extension ([from source](docs/guides/extension-install.md) or a release package).
3. Open Jira (Cloud or Server/DC), click the **Session** chip to capture + validate, then use **Projects** to add a project and start a crawl (or use the CLI below).

## Crawl a project

```sh
./ticketmosquito crawl PROJ
./ticketmosquito crawl "https://your-domain.atlassian.net/browse/PROJ"

# Self-hosted / Server-DC (context path is auto-detected)
./ticketmosquito crawl "https://jira.mycompany.com/jira/browse/ENG"

# Full refresh: re-fetch every issue and re-download attachments
./ticketmosquito crawl --force PROJ
```

Refresh a single issue (always overwrites text; attachments skip unless you crawled with `--force`):

```sh
./ticketmosquito crawl-issue PROJ-123
```

## Search and get

Multi-word queries match **all** terms (AND). Summary matches rank above body-only hits.

```sh
./ticketmosquito search "login bug"
./ticketmosquito search --project PROJ "login bug"
./ticketmosquito search --limit 50 "SSO"

./ticketmosquito get-issue PROJ-123
```

REST:

```sh
curl -s "http://127.0.0.1:11043/api/search?q=login+bug&project=PROJ"
curl -s "http://127.0.0.1:11043/api/issues/PROJ-123"
curl -s "http://127.0.0.1:11043/api/projects"
```

## MCP (agents)

TicketMosquito serves MCP over Streamable HTTP at `http://127.0.0.1:11043/mcp` (same process as `serve`). Tools read the **local** catalog only — see **[Configure MCP](docs/guides/configure-mcp.md)**.

| Tool | Role |
|------|------|
| `jira_search` | FTS search |
| `jira_get_issue` | Full Markdown + metadata by key |
| `jira_list_projects` | List crawled projects |

## Commands

| Command | Description |
|---------|-------------|
| `init` | Create data directory, config, run migrations |
| `serve` | Start HTTP API + MCP server on `:11043` |
| `crawl [--force] <project>` | Crawl a Jira project by key or URL |
| `crawl-issue <KEY>` | Refresh one issue |
| `search "query"` | Full-text search (`--project`, `--limit`) |
| `get-issue <KEY>` | Print issue Markdown + metadata |
| `reindex` | Rebuild FTS index from SQLite rows |
| `version` | Print build version |

## Architecture and decisions

- [Architecture](docs/ARCHITECTURE.md)
- [ADRs](docs/adr/README.md)
- [Changelog](CHANGELOG.md)
- Contributor setup: [dev/README.md](dev/README.md)

## License

See repository license file if present; otherwise treat as private until published.
