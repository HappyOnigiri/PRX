## prx config sync update

Update the automatic GitHub synchronization interval

```
prx config sync update [flags]
```

### Examples

```
prx config sync update --interval-seconds 3600 --json
```

### Options

```
  -h, --help                   help for update
      --interval-seconds int   automatic sync interval in seconds (minimum 600)
```

### Options inherited from parent commands

```
      --config string           YAML configuration path (env: PRX_CONFIG)
      --db string               SQLite database path (env: PRX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
      --json                    output JSON
```

### SEE ALSO

* [prx config sync](prx_config_sync.md)	 - Manage automatic GitHub synchronization settings

