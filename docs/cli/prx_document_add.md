## prx document add

Add a URL or local Markdown path

```
prx document add [flags]
```

### Examples

```
prx document add --task TASK_ID --kind markdown_path --value docs/checkout.md --json
```

### Options

```
      --feature string   feature ID or slug
  -h, --help             help for add
      --kind string      url or markdown_path (default "url")
      --task string      task ID
      --title string     document title
      --value string     URL or Markdown path
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

* [prx document](prx_document.md)	 - Manage URL and local Markdown references

