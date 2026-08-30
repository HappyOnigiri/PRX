## prx task list

List tasks, optionally filtered by feature

```
prx task list [flags]
```

### Examples

```
prx task list --feature checkout --json
```

### Options

```
      --feature string   filter by feature
  -h, --help             help for list
```

### Options inherited from parent commands

```
      --config string           YAML configuration path (env: PRX_CONFIG)
      --db string               SQLite database path (env: PRX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
      --json                    emit compact JSON responses
```

### SEE ALSO

* [prx task](prx_task.md)	 - Manage implementation and manual tasks

