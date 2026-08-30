# PRX

PRX is a local-first dependency control room for initiatives that span many GitHub pull requests. It stores a normalized DAG in SQLite, derives the next safe tasks, refreshes pull-request state directly from GitHub, and exposes the same application rules through a non-interactive CLI and a ConnectRPC-powered React workspace.

## Requirements

- Go 1.26 or newer
- The Node.js version is defined in `.tool-versions`, and the pnpm version is defined in the root `package.json` `packageManager`
- A Chromium browser for Playwright development checks
- A GitHub credential configured in `config.yaml`, or the historical `GITHUB_TOKEN`, `GH_TOKEN`, or authenticated `gh` CLI fallback for GitHub.com synchronization

`sqlc`, Buf, and Protocol Buffer generators are pinned as Go tool dependencies, and `make lint` installs `golangci-lint` into `bin/` at the version in `.tool-versions`. None of them need global installation. The shipped binary does not require Node.js, pnpm, or `gh` when a token is supplied.

## Build and start

```sh
make install
prx seed --github-fixture demo
prx serve
```

`make install` installs the locked web dependencies, builds a local binary identified as `<version>-dev`, and installs it to `~/.local/bin/prx`. Set `INSTALL_DIR` to install elsewhere. Ensure the installation directory is on `PATH`, then open <http://127.0.0.1:7331>. The production web build is embedded in the binary; no separate frontend process is needed.

The default database is stored under the operating system's user configuration directory. Use `--db /path/to/prx.db` or `PRX_DB` to select another database. The server binds only to `127.0.0.1:7331` unless `--addr` is explicitly supplied.

## Develop

```sh
make dev
```

Open <http://127.0.0.1:7331>, the same URL used by the production server. Vite applies WebUI changes with hot module replacement and proxies RPC requests to the development Go server on port 7332. Air rebuilds and restarts the Go server when Go source files change. Stop any existing `prx serve` process that is using port 7331 before starting the development servers.

The development server uses the same live GitHub configuration as `prx serve`; use `prx config` or `GITHUB_TOKEN`, `GH_TOKEN`, or an authenticated `gh` CLI before starting `make dev`. Use `--config /path/to/config.yaml` or `PRX_CONFIG` to select a configuration file.

Run `make ci` before handing off a change.

## Documentation

- `prx -h`, and `-h` on any subcommand, documents the CLI. [docs/cli/prx.md](docs/cli/prx.md) is the same reference in Markdown, generated from the command definitions by `make generate`.
- [docs/design.md](docs/design.md) covers package boundaries, status and dependency rules, storage, the local trust boundary, UI structure, and trade-offs.
- [docs/development.md](docs/development.md) covers the verification targets, coverage contracts, end-to-end tests, and GitHub fixtures.
