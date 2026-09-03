## prx project delete

Delete a project; --cascade removes its documents and releases its features

### Synopsis

Delete a project.

Without --cascade the command fails while the project still has features or documents.

With --cascade it deletes the project's own documents and releases its features.
Contained features are never deleted: they keep their own identifiers and tasks.

```
prx project delete PROJECT_ID_OR_SLUG [flags]
```

### Examples

```
prx project delete payments --cascade
```

### Options

```
      --cascade   delete the project's documents and release its features
  -h, --help      help for delete
```

### Options inherited from parent commands

```
      --config string           YAML configuration path (env: PRX_CONFIG)
      --db string               SQLite database path (env: PRX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
      --json                    output JSON
```

### SEE ALSO

* [prx project](prx_project.md)	 - List projects or show one by ID or slug

