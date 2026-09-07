## prx feature create

Create a feature in a project

```
prx feature create TITLE --project PROJECT_ID [flags]
```

### Examples

```
prx feature create "Checkout rollout" --project P-1
prx feature create -- "-fix checkout" --project P-1
```

### Options

```
      --description string   feature description
  -h, --help                 help for create
      --project string       project ID to join; required
```

### Options inherited from parent commands

```
      --config string           YAML configuration path (env: PRX_CONFIG)
      --db string               SQLite database path (env: PRX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
      --json                    output JSON
```

### SEE ALSO

* [prx feature](prx_feature.md)	 - List features or show one by ID

