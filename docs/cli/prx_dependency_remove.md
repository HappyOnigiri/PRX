## prx dependency remove

Remove a dependency; missing edges return not_found

```
prx dependency remove BLOCKER_TASK_ID BLOCKED_TASK_ID [flags]
```

### Examples

```
prx dependency remove BLOCKER_TASK_ID BLOCKED_TASK_ID --json
```

### Options

```
  -h, --help   help for remove
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

* [prx dependency](prx_dependency.md)	 - List or manage directed blocker edges

