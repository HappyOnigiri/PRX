## prx config host update

Update a configured GitHub host

```
prx config host update HOST [flags]
```

### Examples

```
prx config host update ghe.example.com --api-url https://ghe.example.com/api/v3/ --json
```

### Options

```
      --api-url string      new HTTPS API base URL
  -h, --help                help for update
      --new-host string     new hostname
      --upload-url string   new HTTPS upload base URL
      --web-url string      new HTTPS web URL
```

### Options inherited from parent commands

```
      --config string           YAML configuration path (env: PRX_CONFIG)
      --db string               SQLite database path (env: PRX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
      --json                    emit compact JSON responses
```

### SEE ALSO

* [prx config host](prx_config_host.md)	 - Manage configured GitHub hosts

