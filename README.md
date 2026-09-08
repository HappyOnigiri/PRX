# PRX

[日本語](./README.ja.md)

**PRX** keeps an initiative that is spread across many GitHub pull requests visible from your own machine.

Once an initiative splits into ten or twenty pull requests, it stops being obvious which one is waiting on someone else and which one you can pick up now.
Register how the tasks depend on each other, and PRX separates the ready work from the blocked work for you.

## What it does

- **Shows what you can start**: pull out only the tasks whose dependencies are all satisfied, instead of working out the order again every time.
- **Shows where things are stuck**: look over the whole initiative as a graph and follow which task is waiting on what.
- **Reflects pull-request state**: review status, conflicts, and merges are read from GitHub and folded into how the tasks progress.
- **Suggests the next move**: list the pull requests waiting for review and the tasks that have gone stale.
- **Builds prompts for agents**: assemble a prompt from a template with the task and its dependencies filled in, ready to copy.
- **Stays on your machine**: data is stored locally, and the server accepts only local connections by default.

The browser workspace and the scriptable command line offer the same operations.

## Getting started

```sh
curl -fsSL https://github.com/HappyOnigiri/PRX/releases/latest/download/install.sh | bash
prx daemon install # start PRX at login (macOS)
prx open           # open the running server in a browser
```

The installer downloads the latest release, verifies its checksum, and installs the binary to `~/.local/bin/prx`.
Run the same command again to update; the installed binary is kept when the download or the verification fails.
Releases are built for macOS on Apple Silicon, and the binary carries the browser workspace with it.

To build from a checkout instead, run `make install`, which installs to `~/.local/bin/prx` as well.
Set `INSTALL_DIR` to install it elsewhere.

`prx daemon install` registers a LaunchAgent that runs the server for you from the next login onward, and `prx open` opens whichever address it is actually listening on.
The port defaults to 7331 and falls back to another one when that is taken, so read the address from `prx open` rather than typing it.
`prx config server update PORT` pins a port; run `prx daemon restart` afterwards to move a server that is already running.
`prx serve` still runs the server in the foreground on any operating system.

`prx serve --demo` starts a demo loaded with sample data.
It leaves your own data untouched, so use it to try PRX first.

To synchronize with GitHub, supply a credential through `prx config`, `GITHUB_TOKEN`, `GH_TOKEN`, or an authenticated `gh` CLI.
Tasks and dependencies work the same way without it.

## Develop

```sh
make dev  # start the development server: http://127.0.0.1:7331
make demo # same, backed by isolated demo data
make ci   # run every check before handing off a change
```

`make demo` restarts the API on Go changes, which recreates the demo data from scratch.

Building requires Go 1.27 or newer, plus the Node.js and pnpm versions pinned in `.tool-versions` and `package.json`.
Browser checks during development use Chromium.
Every other tool is pinned in the repository, so none of them need a separate installation.

## Documentation

- **Command usage**: `prx -h`, and `-h` on any subcommand.
- [docs/cli/prx.md](docs/cli/prx.md): the same reference in Markdown.
- [docs/design/](docs/design/README.md): design decisions and the reasoning behind them.
- [docs/development.md](docs/development.md): verification and release rules.
