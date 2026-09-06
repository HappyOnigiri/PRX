## prx feature delete

Delete a feature and optionally its contained data

```
prx feature delete FEATURE_ID [flags]
```

### Examples

```
prx feature delete F-1 --cascade
```

### Options

```
      --cascade   delete contained tasks and references
  -h, --help      help for delete
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

