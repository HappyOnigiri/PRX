## prx document update

Update a document

```
prx document update DOCUMENT_ID [flags]
```

### Examples

```
prx document update DOCUMENT_ID --title Runbook
```

### Options

```
  -h, --help                   help for update
      --implementation-plan    set or clear the plan designation
      --local-file string      registered local file path
      --markdown-file string   read stored Markdown from a file
      --stdin                  read stored Markdown from standard input
      --title string           replace the document title
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

