## prx feature create

Create a feature

```
prx feature create TITLE [flags]
```

### Examples

```
prx feature create "Checkout rollout"
prx feature create "Checkout rollout" --project P-1
prx feature create -- "-fix checkout"
```

### Options

```
      --description string   feature description
  -h, --help                 help for create
      --project string       project ID to join
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

