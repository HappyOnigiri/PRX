## prx document add

Add a URL or local Markdown path

```
prx document add FEATURE_ID_OR_SLUG_OR_TASK_ID VALUE [flags]
```

### Examples

```
prx document add TASK_ID docs/checkout.md --kind markdown_path
```

### Options

```
  -h, --help           help for add
      --kind string    url or markdown_path (default "url")
      --title string   document title
```

### Options inherited from parent commands

```
      --config string           YAML configuration path (env: PRX_CONFIG)
      --db string               SQLite database path (env: PRX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
      --json                    output JSON
```

### SEE ALSO

* [prx document](prx_document.md)	 - List or manage URL and local Markdown references

