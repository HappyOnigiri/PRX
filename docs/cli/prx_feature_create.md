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
      --db string               SQLite database path (env: PRX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
      --json                    emit a stable JSON envelope
```

### SEE ALSO

* [prx feature](prx_feature.md)	 - Manage features

