## prx feature create

Create a feature

```
prx feature create [flags]
```

### Examples

```
prx feature create --slug checkout --title "Checkout rollout" --json
```

### Options

```
      --description string   feature description
  -h, --help                 help for create
      --slug string          stable feature slug
      --title string         feature title
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

* [prx feature](prx_feature.md)	 - Manage features

