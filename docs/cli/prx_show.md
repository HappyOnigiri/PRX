## prx show

Show a project, a feature, or a task by public identifier

### Synopsis

Show a project, a feature, or a task by public identifier.

The operand is a public project, feature, or task ID, or a feature or project slug.

```
prx show PROJECT_OR_FEATURE_OR_TASK [flags]
```

### Examples

```
prx show F-1
prx show checkout
prx show T-1
prx show P-1
```

### Options

```
  -h, --help   help for show
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

