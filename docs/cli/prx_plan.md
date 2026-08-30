## prx plan

Show or manage a task's implementation plan

```
prx plan TASK_ID [flags]
```

### Examples

```
prx plan T-1 --json
```

### Options

```
  -h, --help   help for plan
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
* [prx plan delete](prx_plan_delete.md)	 - Delete a task's implementation plan
* [prx plan set](prx_plan_set.md)	 - Create or replace a task's implementation plan

