## prx sync

Refresh GitHub state for pull-request tasks

```
prx sync [flags]
```

### Examples

```
prx sync --feature FEATURE_ID
prx sync --task TASK_ID
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
      --json                    output JSON
```

### SEE ALSO

* [prx](prx.md)	 - Manage pull-request dependency roadmaps
* [prx sync status](prx_sync_status.md)	 - Show automatic GitHub synchronization status

