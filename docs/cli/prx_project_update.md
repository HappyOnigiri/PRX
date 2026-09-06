## prx project update

Update a project by ID

```
prx project update PROJECT_ID [flags]
```

### Examples

```
prx project update P-1 --title "Payments platform"
```

### Options

```
      --archived             archive (true) or unarchive (false) the project
      --description string   new description
  -h, --help                 help for update
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

* [prx project](prx_project.md)	 - List projects or show one by ID

