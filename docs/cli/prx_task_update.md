## prx task update

Update a task by ID

```
prx task update TASK_ID [flags]
```

### Examples

```
prx task update TASK_ID --status completed --json
```

### Options

```
      --assignee string   new assignee
  -h, --help              help for update
      --scope string      new scope
      --status string     auto, not_started, in_progress, completed, or closed
      --title string      new title
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

* [prx task](prx_task.md)	 - Manage implementation and manual tasks

