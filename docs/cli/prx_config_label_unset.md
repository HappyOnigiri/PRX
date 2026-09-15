## prx config label unset

Remove a global task label text or color override

```
prx config label unset KEY [flags]
```

### Examples

```
prx config label unset status.in_progress --text
```

### Options

```
      --color   remove color override
  -h, --help    help for unset
      --text    remove text override
```

### Options inherited from parent commands

```
      --config string           YAML configuration path (env: PRX_CONFIG)
      --db string               SQLite database path (env: PRX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
      --json                    output JSON
```

### SEE ALSO

* [prx config label](prx_config_label.md)	 - Show or manage global task label overrides

