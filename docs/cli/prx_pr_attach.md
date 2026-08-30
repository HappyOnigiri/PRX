## prx pr attach

Attach a GitHub pull request to a task

```
prx pr attach TASK_ID URL [flags]
```

### Examples

```
prx pr attach TASK_ID https://github.com/acme/payments/pull/42
```

### Options

```
  -h, --help   help for attach
```

### Options inherited from parent commands

```
      --config string           YAML configuration path (env: PRX_CONFIG)
      --db string               SQLite database path (env: PRX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
      --json                    output JSON
```

### SEE ALSO

* [prx pr](prx_pr.md)	 - List or attach GitHub pull requests

