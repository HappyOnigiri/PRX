## prx feature

List features or show one by ID

### Synopsis

List features or show one by ID.

Alias: f.

```
prx feature [FEATURE_ID] [flags]
```

### Examples

```
prx feature
prx feature F-1
prx f F-1
```

### Options

```
  -h, --help   help for feature
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
* [prx feature archive](prx_feature_archive.md)	 - Archive a feature by ID
* [prx feature create](prx_feature_create.md)	 - Create a feature
* [prx feature delete](prx_feature_delete.md)	 - Delete a feature and optionally its contained data
* [prx feature unarchive](prx_feature_unarchive.md)	 - Unarchive a feature by ID
* [prx feature update](prx_feature_update.md)	 - Update a feature by ID

