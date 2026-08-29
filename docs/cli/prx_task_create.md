## prx task create

Create an implementation or manual task

```
prx task create [flags]
```

### Examples

```
prx task create --feature checkout --title "Add payment intent API" --assignee Mika --json
```

### Options

```
      --assignee string   assignee
      --feature string    feature ID or slug
  -h, --help              help for create
      --kind string       pr or manual (default "pr")
      --scope string      scope description
      --title string      task title
```

### Options inherited from parent commands

```
      --db string               SQLite database path (env: PRX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
      --json                    emit a stable JSON envelope
```

### SEE ALSO

* [prx task](prx_task.md)	 - Manage implementation and manual tasks

