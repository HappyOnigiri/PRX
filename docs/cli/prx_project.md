## prx project

List projects or show one by ID or slug

### Synopsis

List projects or show one by ID or slug.

Alias: proj.

```
prx project [PROJECT_ID_OR_SLUG] [flags]
```

### Examples

```
prx project
prx project P-1
prx proj payments
```

### Options

```
  -h, --help   help for project
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
* [prx project archive](prx_project_archive.md)	 - Archive a project and make its features read-only
* [prx project create](prx_project_create.md)	 - Create a project
* [prx project delete](prx_project_delete.md)	 - Delete a project; --cascade removes its documents and releases its features
* [prx project unarchive](prx_project_unarchive.md)	 - Unarchive a project and let its features accept writes again
* [prx project update](prx_project_update.md)	 - Update a project by ID or slug

