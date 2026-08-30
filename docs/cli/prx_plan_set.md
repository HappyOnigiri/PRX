## prx plan set

Create or replace a task's implementation plan

### Synopsis

Create or replace a task's implementation plan.

FILE_OR_DASH is a Markdown file path, or - to read the plan content from standard input.

```
prx plan set TASK_ID FILE_OR_DASH [flags]
```

### Examples

```
prx plan set TASK_ID plan.md
prx plan set TASK_ID -
```

### Options

```
  -h, --help   help for set
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

