## prx implementation-plan set

Create or replace a task's implementation plan

```
prx implementation-plan set TASK_ID [flags]
```

### Examples

```
prx implementation-plan set TASK_ID --file plan.md --json
```

### Options

```
      --file string   read plan content from a file
  -h, --help          help for set
      --stdin         read plan content from standard input
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

* [prx implementation-plan](prx_implementation-plan.md)	 - Manage task implementation plans

