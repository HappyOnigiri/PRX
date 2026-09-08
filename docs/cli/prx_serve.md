## prx serve

Start the local WebUI and ConnectRPC server

### Synopsis

Start the local WebUI and ConnectRPC server.

Without --addr the port comes from server.port in the configuration, which defaults to 7331 and falls back to an ephemeral port when that is in use.
--addr is the only way to listen outside loopback, and such a server is not recorded as the running one.

```
prx serve [flags]
```

### Examples

```
prx serve --demo
```

### Options

```
      --addr string   listen address, overriding server.port and loopback (host:port)
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

