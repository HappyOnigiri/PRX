## prx dependency

List or manage directed blocker edges

### Synopsis

List or manage directed blocker edges.

Alias: dep.

```
prx dependency [flags]
```

### Examples

```
prx dependency --json
prx dep --json
```

### Options

```
  -h, --help   help for dependency
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

* [prx](prx.md)	 - Manage pull-request dependency roadmaps
* [prx dependency add](prx_dependency_add.md)	 - Add a blocker-to-blocked dependency
* [prx dependency remove](prx_dependency_remove.md)	 - Remove a dependency; missing edges return not_found

