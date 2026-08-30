## prx document delete

Delete a document; missing documents return not_found

```
prx document delete DOCUMENT_ID [flags]
```

### Examples

```
prx document delete DOCUMENT_ID
```

### Options

```
  -h, --help   help for delete
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

