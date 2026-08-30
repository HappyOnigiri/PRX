## prx

Manage pull-request dependency roadmaps

### Options

```
      --config string           YAML configuration path (env: PRX_CONFIG)
      --db string               SQLite database path (env: PRX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
  -h, --help                    help for prx
      --json                    emit a stable JSON envelope
  -v, --version                 version for prx
```

### SEE ALSO

* [prx config](prx_config.md)	 - Manage GitHub hosts and authentication
* [prx conflicts](prx_conflicts.md)	 - List tasks with conflicting pull requests
* [prx dependency](prx_dependency.md)	 - Manage directed blocker edges
* [prx document](prx_document.md)	 - Manage URL and local Markdown references
* [prx feature](prx_feature.md)	 - Manage features
* [prx graph](prx_graph.md)	 - Show a feature graph with tasks and dependencies
* [prx pr](prx_pr.md)	 - Attach GitHub pull requests
* [prx ready](prx_ready.md)	 - List tasks whose blockers are satisfied
* [prx reviews](prx_reviews.md)	 - List tasks waiting for pull-request reviews
* [prx seed](prx_seed.md)	 - Create deterministic demo roadmap data
* [prx serve](prx_serve.md)	 - Start the local WebUI and ConnectRPC server
* [prx snapshot](prx_snapshot.md)	 - Show the complete current snapshot
* [prx stale](prx_stale.md)	 - List tasks with stale GitHub state
* [prx sync](prx_sync.md)	 - Refresh GitHub state for pull-request tasks
* [prx task](prx_task.md)	 - Manage implementation and manual tasks
* [prx validate](prx_validate.md)	 - Validate the stored dependency data

