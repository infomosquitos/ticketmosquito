# Install from source

Build the `ticketmosquito` binary and browser extensions from a git clone.

For pre-built binaries (when published), see [Install](INSTALL.md).

## Requirements

- Go 1.25+ (tested with 1.26)
- npm for building extensions
- Firefox or Chrome (for the Ticket Mosquito extension)

## Binary

```sh
git clone <repo-url> ticketmosquito
cd ticketmosquito
make build
# binary: ./ticketmosquito
```

Cross-compile example:

```sh
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -ldflags "-s -w -X github.com/infomosquitos/ticketmosquito/internal/cliapp.Version=dev" \
  -o ticketmosquito-linux-amd64 \
  ./cmd/ticketmosquito
```

Release-style artifacts (binaries + extension zips):

```sh
./scripts/build-release.sh v0.1.0
ls dist/
```

## Browser extensions

```sh
make build-extensions
```

See [Extension install](guides/extension-install.md) for loading Chrome unpacked / Firefox temporary add-on.

## First run

```sh
./ticketmosquito init
./ticketmosquito serve
```

Then capture a Jira session with the extension and crawl a project — see the [README](../README.md).

Contributor workflows and tests: [dev/README.md](../dev/README.md).
