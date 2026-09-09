# PRX

English | [日本語](README.ja.md) | [简体中文](README.zh-CN.md)

**PRX** keeps an initiative that is spread across many GitHub pull requests visible from your own machine.
Register how the tasks depend on each other, and PRX separates the work you can start now from the work that is waiting on something.

## Features

- **Shows what you can start** — Pull out only the tasks whose dependencies are all satisfied, instead of working out the order again every time.
- **Shows where things are stuck** — Look over the whole initiative as a graph and follow which task is waiting on what, including pull requests awaiting review and tasks that have gone stale.
- **Reflects pull-request state** — Review status, conflicts, and merges are read from GitHub and folded into how the tasks progress.
- **Builds prompts for agents** — Assemble a prompt from a template with the task and its dependencies filled in, ready to copy.
- **Stays on your machine** — Data is stored locally, and the server accepts only local connections by default.

## Installation

### macOS

A prebuilt binary is published for Apple Silicon.

```sh
curl -fsSL https://github.com/HappyOnigiri/PRX/releases/latest/download/install.sh | bash
```

Run the same command again to update.

### Linux / WSL2

Build from source; the same steps work on macOS. Windows is not supported natively.

```sh
git clone https://github.com/HappyOnigiri/PRX.git
cd PRX
make install
```

`make install` builds the WebUI along with the binary and installs it to `~/.local/bin/prx`. Set `INSTALL_DIR` to install it elsewhere.
`prx daemon` and `prx open` manage the background service on macOS only, so start the server with `prx serve` instead.

## Usage

Open http://localhost:7331/ in a browser. When that port is taken and the server moved to another one, `prx open` opens whichever address it is actually listening on.

The `prx` command reads and writes the same data, so an AI agent can look at the current state and register tasks and dependencies.

```sh
prx ready      # pull out the tasks you can start
prx graph F-1  # see a whole initiative with its tasks and dependencies
prx prompt T-1 # assemble the prompt to hand to a task
```

See `prx -h` and `prx <command> -h` for commands and options.

## More options

- **Synchronize with GitHub:** supply a credential through `prx config`, `GITHUB_TOKEN`, `GH_TOKEN`, or an authenticated `gh` CLI. Tasks and dependencies work the same way without it.
- **Pin a port:** `prx config server update PORT`. Run `prx daemon restart` afterwards to move a server that is already running.
- **Run in the foreground:** `prx serve` runs the server in the foreground on any operating system.
- **Demo:** `prx serve --demo` starts a demo loaded with sample data. It leaves your own data untouched, so use it to try PRX first.

## Uninstallation

```sh
curl -fsSL https://github.com/HappyOnigiri/PRX/releases/latest/download/uninstall.sh | bash
```

## Development

```sh
make dev  # start the development server: http://127.0.0.1:7331
make demo # same, backed by isolated demo data
make ci   # run every check before handing off a change
```

`make demo` restarts the API on Go changes, which recreates the demo data from scratch.

## Documentation

- [docs/cli/prx.md](docs/cli/prx.md): the CLI reference in Markdown.
- [docs/design/](docs/design/README.md): design decisions and the reasoning behind them (Japanese).
- [docs/development.md](docs/development.md): verification and release rules (Japanese).

## Contributing

Contributions are welcome!
Share bug reports and ideas through [Issues](https://github.com/HappyOnigiri/PRX/issues), or send a [pull request](https://github.com/HappyOnigiri/PRX/pulls).
Documentation improvements and translations are welcome too.
