## prx config host remove

Remove a configured GitHub host

```
prx config host remove HOST [flags]
```

### Examples

```
prx config host remove ghe.example.com --json
```

### Options

```
  -h, --help   help for remove
```

### Options inherited from parent commands

```
      --config string           YAML configuration path (env: PRX_CONFIG)
      --db string               SQLite database path (env: PRX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
      --human                   force human-readable responses
      --json                    force compact JSON responses
```

### SEE ALSO

* [prx config host](prx_config_host.md)	 - List or manage configured GitHub hosts

