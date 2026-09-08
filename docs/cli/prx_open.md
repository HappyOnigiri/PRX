## prx open

Open the running PRX WebUI in a browser

### Synopsis

Open the running PRX WebUI in a browser.

The URL comes from the running server itself, so it is correct even when the port fell back to an ephemeral one. This never starts a server; use prx daemon start for that.

```
prx open [flags]
```

### Examples

```
prx open --print
```

### Options

```
  -h, --help    help for open
      --print   print the URL instead of opening a browser
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

