## prx config auth update

Update a host-scoped authentication method

```
prx config auth update AUTH_METHOD_ID [flags]
```

### Examples

```
prx config auth update work-gh --user HappyOnigiri --json
```

### Options

```
      --account string    Keychain account
  -h, --help              help for update
      --host string       configured host
      --new-id string     new authentication method ID
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
      --human                   force human-readable responses
      --json                    force compact JSON responses
```

### SEE ALSO

* [prx config auth](prx_config_auth.md)	 - List or manage host-scoped authentication methods

