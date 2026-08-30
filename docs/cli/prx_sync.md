## prx sync

Refresh GitHub state for pull-request tasks

```
prx sync [flags]
```

### Examples

```
prx sync --feature checkout --json
```

### Options

```
      --feature string   feature ID or slug
  -h, --help             help for sync
      --task string      task ID
```

### Options inherited from parent commands

```
      --config string           YAML configuration path (env: PRX_CONFIG)
      --db string               SQLite database path (env: PRX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
      --json                    emit compact JSON responses
```

### SEE ALSO

* [prx](prx.md)	 - Manage pull-request dependency roadmaps

