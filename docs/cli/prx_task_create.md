## prx task create

Create an implementation or manual task

```
prx task create FEATURE_ID_OR_SLUG TITLE [flags]
```

### Examples

```
prx task create checkout "Add payment intent API" --assignee Mika
```

### Options

```
      --assignee string   assignee
  -h, --help              help for create
      --kind string       pr or manual (default "pr")
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

