## prx seed

Create deterministic demo roadmap data

```
prx seed [flags]
```

### Examples

```
prx seed --github-fixture demo --features 100 --tasks 50 --json
```

### Options

```
      --features int   number of demo features (default 1)
  -h, --help           help for seed
      --slug string    feature slug (default "demo-roadmap")
      --tasks int      number of demo tasks (default 8)
```

### Options inherited from parent commands

```
      --db string               SQLite database path (env: PRX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
      --json                    emit a stable JSON envelope
```

### SEE ALSO

* [prx](prx.md)	 - Manage pull-request dependency roadmaps

