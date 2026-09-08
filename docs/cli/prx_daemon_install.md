## prx daemon install

Register the LaunchAgent that starts PRX at login

### Synopsis

Register the LaunchAgent that starts PRX at login.

launchd starts the server right away, so the command waits until that server is listening and reports its address.

```
prx daemon install [flags]
```

### Examples

```
prx daemon install
```

### Options

```
  -h, --help   help for install
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

