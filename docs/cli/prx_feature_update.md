## prx feature update

Update a feature by ID or slug

```
prx feature update FEATURE_ID_OR_SLUG [flags]
```

### Examples

```
prx feature update checkout --archived=false --json
```

### Options

```
      --archived             archive (true) or unarchive (false) the feature
      --description string   new description
  -h, --help                 help for update
      --slug string          new slug
      --status string        active, paused, completed, or cancelled
      --title string         new title
```

### Options inherited from parent commands

```
      --config string           YAML configuration path (env: PRX_CONFIG)
      --db string               SQLite database path (env: PRX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
      --json                    emit compact JSON responses
```

### SEE ALSO

* [prx feature](prx_feature.md)	 - Manage features

