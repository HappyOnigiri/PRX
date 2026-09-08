## prx daemon

Show or manage the background PRX server

### Synopsis

Show or manage the background PRX server.

On macOS a LaunchAgent starts prx serve at login. The LaunchAgent never passes --addr or --demo, so the background server always listens on loopback with real data.
Other operating systems report daemon_unsupported; run prx serve directly there.

```
prx daemon [flags]
```

### Examples

```
prx daemon
```

### Options

```
  -h, --help   help for daemon
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
* [prx daemon install](prx_daemon_install.md)	 - Register the LaunchAgent that starts PRX at login
* [prx daemon restart](prx_daemon_restart.md)	 - Replace the background server with a fresh one
* [prx daemon start](prx_daemon_start.md)	 - Ask launchd to start the background server
* [prx daemon stop](prx_daemon_stop.md)	 - Stop the background server without removing the LaunchAgent
* [prx daemon uninstall](prx_daemon_uninstall.md)	 - Remove the LaunchAgent and stop starting PRX at login

