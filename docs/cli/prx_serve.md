## prx serve

Start the local WebUI and ConnectRPC server

```
prx serve [flags]
```

### Examples

```
prx serve --demo
```

### Options

```
      --addr string   listen address (default "127.0.0.1:7331")
      --demo          start with isolated temporary demo data
  -h, --help          help for serve
```

### Options inherited from parent commands

```
      --config string           YAML configuration path (env: PRX_CONFIG)
      --db string               SQLite database path (env: PRX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
      --json                    output JSON
```

### SEE ALSO

* [prx](prx.md)	 - Manage pull-request dependency roadmaps

