## prx document add

Add a document

```
prx document add [flags]
```

### Examples

```
prx document add --task T-1 --url https://example.com
prx document add --feature F-1 --markdown-file notes.md
```

### Options

```
      --feature string         feature ID or slug
  -h, --help                   help for add
      --implementation-plan    mark as the task implementation plan
      --kind string            compatibility alias: url, local_file, or markdown_path (default "url")
      --local-file string      registered local file path
      --markdown-file string   read stored Markdown from a file
      --stdin                  read stored Markdown from standard input
      --task string            task ID
      --title string           document title
      --url string             HTTP or HTTPS URL
      --value string           compatibility value for --kind
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

