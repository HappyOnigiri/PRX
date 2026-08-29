## prx task delete

Delete a task and optionally its dependencies and references

```
prx task delete TASK_ID [flags]
```

### Examples

```
prx task delete TASK_ID --cascade --json
```

### Options

```
      --cascade   delete dependencies and references
  -h, --help      help for delete
```

### Options inherited from parent commands

```
      --db string               SQLite database path (env: PRX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
      --json                    emit a stable JSON envelope
```

### SEE ALSO

* [prx task](prx_task.md)	 - Manage implementation and manual tasks

