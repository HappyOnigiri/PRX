## prx pr detach

Detach a pull request; missing tasks return not_found

```
prx pr detach TASK_ID [flags]
```

### Examples

```
prx pr detach TASK_ID --json
```

### Options

```
  -h, --help   help for detach
```

### Options inherited from parent commands

```
      --db string               SQLite database path (env: PRX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
      --json                    emit a stable JSON envelope
```

### SEE ALSO

* [prx pr](prx_pr.md)	 - Attach GitHub pull requests

