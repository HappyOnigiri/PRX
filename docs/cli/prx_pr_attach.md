## prx pr attach

Attach a GitHub pull request to a task

```
prx pr attach [flags]
```

### Examples

```
prx pr attach --task TASK_ID --url https://github.com/acme/payments/pull/42 --json
```

### Options

```
  -h, --help          help for attach
      --task string   task ID
      --url string    GitHub pull request URL
```

### Options inherited from parent commands

```
      --db string               SQLite database path (env: PRX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
      --json                    emit a stable JSON envelope
```

### SEE ALSO

* [prx pr](prx_pr.md)	 - Attach GitHub pull requests

