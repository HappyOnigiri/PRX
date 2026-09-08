## prx daemon restart

Replace the background server with a fresh one

### Synopsis

Replace the background server with a fresh one.

Use this after installing a new PRX binary so the server stops answering with an older embedded schema.

```
prx daemon restart [flags]
```

### Examples

```
prx daemon restart
```

### Options

```
  -h, --help   help for restart
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

