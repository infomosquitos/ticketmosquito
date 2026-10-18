# Release runbook

Related: [CHANGELOG.md](../CHANGELOG.md), [CLEANUP.md](CLEANUP.md), [INSTALL.md](../docs/INSTALL.md)

## When to release

Cut a release when:

- [CHANGELOG.md](../CHANGELOG.md) `[Unreleased]` contains a significant work chunk.
- `make test` (and CI on the default branch) are green.

## Pre-release checklist

- [ ] `go test ./...` passes.
- [ ] `make build` succeeds.
- [ ] Optional: `./scripts/build-release.sh vX.Y.Z` produces `dist/` artifacts.
- [ ] Smoke: `init` → `serve` → extension session → `crawl` → `search` / MCP.

## Finalize the changelog

Edit [CHANGELOG.md](../CHANGELOG.md):

1. Review `[Unreleased]` — group under **Added** / **Changed** / **Removed** / **Fixed**.
2. Add a dated section, e.g. `## [0.1.0] - YYYY-MM-DD`.
3. Move items from `[Unreleased]` into that section.
4. Leave `[Unreleased]` empty for the next cycle.

Optionally promote finished task docs via [CLEANUP.md](CLEANUP.md).

## Execute the release

1. Commit changelog (and any version bumps).
2. Tag and push — CI [`.github/workflows/release.yml`](../.github/workflows/release.yml) builds binaries + extension zips and publishes a GitHub Release:

```sh
git commit -m "Release X.Y.Z"
git tag -a vX.Y.Z -m "Release X.Y.Z"
git push origin HEAD --tags
```

Local dry-run without tagging:

```sh
./scripts/build-release.sh vX.Y.Z
ls dist/
```

Version is stamped into the binary via `-ldflags` → `internal/cliapp.Version`.
