## prx config server update

Update the local server listen port

### Synopsis

Update the local server listen port.

PORT is "auto" or a port number between 1 and 65535. With "auto", PRX prefers 7331 and falls back to an ephemeral port when it is already in use.
The host is always loopback; use prx serve --addr to listen elsewhere.
A server that is already running keeps its current port until it is restarted; run prx daemon restart to move the background server.

```
prx config server update PORT [flags]
```

### Examples

```
prx config server update 7400
```

### Options

```
  -h, --help   help for update
```

### Options inherited from parent commands

```
      --config string           YAML configuration path (env: PRX_CONFIG)
      --db string               SQLite database path (env: PRX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
      --json                    output JSON
```

### SEE ALSO

* [prx config server](prx_config_server.md)	 - Show or manage the local server listen port

