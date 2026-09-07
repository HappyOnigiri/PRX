## prx project delete

Delete a project; --cascade removes its documents and the features it holds

### Synopsis

Delete a project.

Without --cascade the command fails while the project still has features or documents.

With --cascade it deletes the project's own documents and every feature inside it,
together with the tasks, dependencies, pull-request attachments, and documents those
features own. A feature cannot outlive its project, because it belongs to one.

```
prx project delete PROJECT_ID [flags]
```

### Examples

```
prx project delete P-1 --cascade
```

### Options

```
      --cascade   delete the project's documents and the features it holds
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

* [prx project](prx_project.md)	 - List projects or show one by ID

