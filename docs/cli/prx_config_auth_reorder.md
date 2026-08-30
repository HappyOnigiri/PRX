## prx config auth reorder

Set authentication priority order

```
prx config auth reorder AUTH_METHOD_ID... [flags]
```

### Examples

```
prx config auth reorder ghe-environment ghe-cli
```

### Options

```
  -h, --help   help for reorder
```

### Options inherited from parent commands

```
      --config string           YAML configuration path (env: PRX_CONFIG)
      --db string               SQLite database path (env: PRX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
      --json                    output JSON
```

### SEE ALSO

* [prx config auth](prx_config_auth.md)	 - List or manage host-scoped authentication methods

