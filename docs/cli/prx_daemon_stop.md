## prx daemon stop

Stop the background server without removing the LaunchAgent

### Synopsis

Stop the background server without removing the LaunchAgent.

PRX sends SIGTERM to the recorded process; launchctl bootout is not used because it would unregister the LaunchAgent until the next login.

```
prx daemon stop [flags]
```

### Examples

```
prx daemon stop
```

### Options

```
  -h, --help   help for stop
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

