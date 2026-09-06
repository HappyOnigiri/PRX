## prx task create

Create a task

```
prx task create FEATURE_ID TITLE [flags]
```

### Examples

```
prx task create F-1 "Add payment intent API" --assignee Bob
prx task create F-1 -- "-fix login redirect"
```

### Options

```
      --assignee string   assignee
  -h, --help              help for create
      --scope string      scope description
```

### Options inherited from parent commands

```
      --config string           YAML configuration path (env: PRX_CONFIG)
      --db string               SQLite database path (env: PRX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
      --json                    output JSON
```

### SEE ALSO

* [prx task](prx_task.md)	 - List tasks or show one by ID

