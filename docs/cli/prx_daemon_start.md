## prx daemon start

Ask launchd to start the background server

### Synopsis

Ask launchd to start the background server.

Starting an already running server succeeds without replacing it.

```
prx daemon start [flags]
```

### Examples

```
prx daemon start
```

### Options

```
  -h, --help   help for start
```

### Options inherited from parent commands

```
      --config string           YAML configuration path (env: PRX_CONFIG)
      --db string               SQLite database path (env: PRX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
      --json                    output JSON
```

### SEE ALSO

* [prx daemon](prx_daemon.md)	 - Show or manage the background PRX server

