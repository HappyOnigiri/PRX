## prx config auth

List or manage host-scoped authentication methods

```
prx config auth [flags]
```

### Examples

```
prx config auth
```

### Options

```
  -h, --help   help for auth
```

### Options inherited from parent commands

```
      --config string           YAML configuration path (env: PRX_CONFIG)
      --db string               SQLite database path (env: PRX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
      --json                    output JSON
```

### SEE ALSO

* [prx config](prx_config.md)	 - Show or manage GitHub hosts and authentication
* [prx config auth add](prx_config_auth_add.md)	 - Add a host-scoped authentication method
* [prx config auth remove](prx_config_auth_remove.md)	 - Remove an authentication method and its cached use
* [prx config auth reorder](prx_config_auth_reorder.md)	 - Set authentication priority order
* [prx config auth update](prx_config_auth_update.md)	 - Update a host-scoped authentication method

