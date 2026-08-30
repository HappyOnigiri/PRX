## prx config auth reorder

Set authentication priority order

```
prx config auth reorder AUTH_METHOD_ID... [flags]
```

### Examples

```
prx config auth reorder ghe-environment ghe-cli --json
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
      --json                    emit compact JSON responses
```

### SEE ALSO

* [prx config auth](prx_config_auth.md)	 - Manage host-scoped authentication methods

