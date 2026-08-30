## prx task

List tasks or show one by ID

### Synopsis

List tasks or show one by ID.

Alias: t.

```
prx task [TASK_ID] [flags]
```

### Examples

```
prx task
prx task --feature checkout
prx task T-1
prx t T-1
```

### Options

```
      --feature string   filter by feature ID or slug
  -h, --help             help for task
```

### Options inherited from parent commands

```
      --config string           YAML configuration path (env: PRX_CONFIG)
      --db string               SQLite database path (env: PRX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
      --json                    output JSON
```

### SEE ALSO

* [prx](prx.md)	 - Manage pull-request dependency roadmaps
* [prx task create](prx_task_create.md)	 - Create an implementation or manual task
* [prx task delete](prx_task_delete.md)	 - Delete a task and optionally its dependencies and references
* [prx task update](prx_task_update.md)	 - Update a task by ID

