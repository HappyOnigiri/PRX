## prx dependency add

Add a blocker-to-blocked dependency

```
prx dependency add BLOCKER_TASK_ID BLOCKED_TASK_ID [flags]
```

### Examples

```
prx dependency add BLOCKER_TASK_ID BLOCKED_TASK_ID
```

### Options

```
  -h, --help   help for add
```

### Options inherited from parent commands

```
      --config string           YAML configuration path (env: PRX_CONFIG)
      --db string               SQLite database path (env: PRX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
      --json                    output JSON
```

### SEE ALSO

* [prx dependency](prx_dependency.md)	 - List or manage directed blocker edges

