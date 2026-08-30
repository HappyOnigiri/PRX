## prx config host

List or manage configured GitHub hosts

```
prx config host [flags]
```

### Examples

```
prx config host --json
```

### Options

```
  -h, --help   help for host
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

* [prx config](prx_config.md)	 - Show or manage GitHub hosts and authentication
* [prx config host add](prx_config_host_add.md)	 - Add a GitHub.com or Enterprise host
* [prx config host remove](prx_config_host_remove.md)	 - Remove a configured GitHub host
* [prx config host update](prx_config_host_update.md)	 - Update a configured GitHub host

