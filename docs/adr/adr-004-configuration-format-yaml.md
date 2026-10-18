# ADR-004: Configuration format — YAML

- **Status:** Accepted
- **Date:** 2026-10-18
- **Related:** SpaceMosquito ADR-008; [ADR-002](adr-002-sqlite-only-distribution.md)

## Context

TicketMosquito needs a single readable config for database path, storage base path, session encryption key, and MCP listen host/port.

## Decision

Use **YAML** (`gopkg.in/yaml.v3`) at `{data-dir}/config.yaml` (default `~/.ticketmosquito/config.yaml`).

| Topic | Choice |
|-------|--------|
| Format | YAML |
| Location | Data directory (not XDG-only) for portable `--data-dir` |
| Secrets | `session.encryption_key` lives in config; file should stay private |

## Alternatives considered

| Option | Why not |
|--------|---------|
| TOML | Fine, but SpaceMosquito / selective paste already YAML |
| JSON | No comments; noisy for humans |
| Env-only | Awkward for nested paths and first-run generation |

## Consequences

- `ticketmosquito init` generates config + encryption key.
- Override data dir via `TICKETMOSQUITO_DATA_DIR` / `--data-dir`; optional `CONFIG_PATH`.
