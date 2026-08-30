## prx config auth add

Add a host-scoped authentication method

```
prx config auth add [flags]
```

### Examples

```
prx config auth add --id work-gh --host github.com --type gh_cli
```

### Options

```
      --account string    Keychain account
  -h, --help              help for add
      --host string       configured host
      --id string         authentication method ID
      --service string    Keychain service
      --token-stdin       read an inline token from stdin
      --type string       keychain, environment, inline, or gh_cli
      --user string       gh CLI user
      --variable string   environment variable name
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

