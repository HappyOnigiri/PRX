## prx plan set

Create or replace a task's implementation plan document

```
prx plan set TASK_ID [flags]
```

### Examples

```
prx plan set T-1 --file plan.md
prx plan set T-1 --url https://example.com/plan
```

### Options

```
      --file string         read stored Markdown content from a file
  -h, --help                help for set
      --local-file string   store a local plan path
      --stdin               read stored Markdown content from standard input
      --url string          store an HTTP or HTTPS plan URL
```

### Options inherited from parent commands

```
      --config string           YAML configuration path (env: PRX_CONFIG)
      --db string               SQLite database path (env: PRX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
      --json                    output JSON
```

### SEE ALSO

* [prx plan](prx_plan.md)	 - Show or manage a task's implementation plan document

