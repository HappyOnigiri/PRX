## prx show

Show a feature or task by public identifier

```
prx show FEATURE_ID_OR_SLUG_OR_TASK_ID [flags]
```

### Examples

```
prx show F-1 --json
prx show checkout --json
prx show T-1 --json
```

### Options

```
  -h, --help   help for show
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

* [prx](prx.md)	 - Manage pull-request dependency roadmaps

