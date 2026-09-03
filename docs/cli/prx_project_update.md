## prx project update

Update a project by ID or slug

```
prx project update PROJECT_ID_OR_SLUG [flags]
```

### Examples

```
prx project update payments --title "Payments platform"
```

### Options

```
      --archived             archive (true) or unarchive (false) the project
      --description string   new description
  -h, --help                 help for update
      --slug string          new slug
      --title string         new title
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

