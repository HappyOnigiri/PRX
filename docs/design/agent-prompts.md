# Agent prompt policy

PRX hands tasks to agents that do not know PRX, so prompts must carry task identifiers and the commands needed to act on them.
Rendering task or batch prompts changes no task state, readiness, dependencies, or implementation plan.
Like other read commands, `prx prompt TASK_ID` checks the shared GitHub refresh interval under [github-sync.md](github-sync.md).

## Task prompts

Only the presence of an implementation plan selects the task template; display state and readiness do not.

| Implementation plan | Template | Status timing | Final action |
|---|---|---|---|
| Absent | Design | Set `designing` before anything else | Register a plan, leaving the stored status unchanged |
| Present | Implementation | Set `in_progress` before any change | Record the result |

Setting status before work makes ongoing work visible; registering the plan then presents a designing task as designed.
The built-in implementation and batch templates direct agents to branch from the work's base, including an open blocker's branch for stacked pull requests.

## Batch prompts

The WebUI explicitly requests a batch template for a feature and selected tasks.
The built-in template delegates each task to its own SubAgent, which obtains its instructions through `prx prompt TASK_ID` instead of duplicated task text in the batch body.

Every selected task must belong to the named feature, and every unsatisfied blocker must accompany its blocked task; otherwise rendering fails.
The receiving agent implements blockers first and stacks dependent pull requests on them.
The WebUI offers designed, ready tasks by default and offers blocked tasks on request, leaving selection to the user.
It excludes tasks whose blockers cannot be included and tasks with multiple unsatisfied blockers, since a pull request needs a single base.
Such tasks are handed over individually once their blockers have landed.

## Template configuration and rendering

Templates are shared configuration so the CLI and WebUI emit the same text.
All templates are written together to avoid a partially updated set.
Omitted or blank templates restore built-in defaults so older configuration files continue to load.
Templates matching built-in defaults are omitted from the file so uncustomized installations follow wording updates after upgrades.

Templates use plain substitution over separate, closed vocabularies for task and batch prompts, defined in `internal/prompt/prompt.go`.
Unknown placeholders are rejected; task templates require `{{task_id}}` and batch templates require `{{task_list}}` so every target is identified.
Plan bodies are excluded: they may reach 1 MiB or live behind a locator, so prompts direct agents to `prx plan TASK_ID`.
An absent scope renders as `(not specified)` to distinguish it from a value that failed to load.
The server supplies the vocabulary and built-in defaults alongside stored templates so the editor validates and restores what its server accepts.

The WebUI copies a freshly rendered prompt so a plan registered or deleted since its snapshot cannot select the wrong template.
`prx prompt TASK_ID` prints only the prompt body, ready to use.
Diagnostics report each template's length and whether it matches its built-in default, identifying customization without exposing user-authored text.
