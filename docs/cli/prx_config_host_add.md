## prx config host add

Add a GitHub.com or Enterprise host

```
prx config host add [flags]
```

### Examples

```
prx config host add --host ghe.example.com --json
```

### Options

```
      --api-url string       HTTPS API base URL (defaults from host)
      --graphql-url string   HTTPS GraphQL URL (defaults from host)
  -h, --help                 help for add
      --host string          hostname with optional port
      --upload-url string    HTTPS upload base URL (defaults from host)
      --web-url string       HTTPS web URL (defaults from host)
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

