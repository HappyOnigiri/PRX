## prx document add

Add a document

```
prx document add FEATURE_ID_OR_SLUG_OR_TASK_ID [flags]
```

### Examples

```
prx document add T-1 --url https://example.com
prx document add checkout --markdown-file notes.md
```

### Options

```
  -h, --help                   help for add
      --implementation-plan    mark as the task implementation plan
      --local-file string      registered local file path
      --markdown-file string   read stored Markdown from a file
      --stdin                  read stored Markdown from standard input
      --title string           document title
      --url string             HTTP or HTTPS URL
```

### Options inherited from parent commands

```
      --config string           YAML configuration path (env: PRX_CONFIG)
      --db string               SQLite database path (env: PRX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
      --json                    output JSON
```

### SEE ALSO

* [prx document](prx_document.md)	 - List or manage documents

