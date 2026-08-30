## prx plan set

Create or replace a task's implementation plan

```
prx plan set TASK_ID [flags]
```

### Examples

```
prx plan set TASK_ID --file plan.md
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
      --json                    output JSON
```

### SEE ALSO

* [prx plan](prx_plan.md)	 - Show or manage a task's implementation plan

