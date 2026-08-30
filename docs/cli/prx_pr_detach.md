## prx pr detach

Detach a pull request; missing tasks return not_found

```
prx pr detach TASK_ID [flags]
```

### Examples

```
prx pr detach TASK_ID --json
```

### Options

```
  -h, --help   help for detach
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

* [prx pr](prx_pr.md)	 - List or attach GitHub pull requests

