## prx document

List or manage documents

### Synopsis

List or manage URL, local file, and stored Markdown documents.

Alias: doc.

```
prx document [flags]
```

### Examples

```
prx document
prx document --task T-1
prx doc
```

### Options

```
      --feature string   filter by feature ID or slug
  -h, --help             help for document
      --task string      filter by task ID
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
* [prx document add](prx_document_add.md)	 - Add a document
* [prx document delete](prx_document_delete.md)	 - Delete a document; missing documents return not_found
* [prx document get](prx_document_get.md)	 - Get one document
* [prx document update](prx_document_update.md)	 - Update a document

