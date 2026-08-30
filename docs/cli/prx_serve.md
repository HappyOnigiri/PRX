## prx serve

Start the local WebUI and ConnectRPC server

```
prx serve [flags]
```

### Examples

```
prx serve --addr 127.0.0.1:7331
```

### Options

```
      --addr string   listen address (default "127.0.0.1:7331")
  -h, --help          help for serve
```

### Options inherited from parent commands

```
      --config string           YAML configuration path (env: PRX_CONFIG)
      --db string               SQLite database path (env: PRX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
      --json                    emit a stable JSON envelope
```

### SEE ALSO

* [prx](prx.md)	 - Manage pull-request dependency roadmaps

