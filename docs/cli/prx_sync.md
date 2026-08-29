## prx sync

Refresh GitHub state for pull-request tasks

```
prx sync [flags]
```

### Examples

```
prx sync --feature FEATURE_ID --json
prx sync --task TASK_ID --json
```

### Options

```
      --feature string   feature ID or slug
  -h, --help             help for sync
      --task string      task ID
```

### Options inherited from parent commands

```
      --db string               SQLite database path (env: PRX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
      --json                    emit a stable JSON envelope
```

### SEE ALSO

* [prx](prx.md)	 - Manage pull-request dependency roadmaps

