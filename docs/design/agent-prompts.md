# Agent prompt policy

PRX hands one task to an AI agent that does not know PRX, so the prompt has to carry the task, its identifiers, and the commands the agent needs.
Rendering a prompt never changes task state, readiness, dependencies, or the implementation plan.
`prx prompt TASK_ID` is still an ordinary read command, so it shares the expired-interval GitHub refresh every other read runs.

Which prompt a task gets is derived from one fact only.
A task without an implementation plan gets the design prompt, which ends by registering a plan; a task with one gets the implementation prompt, which ends by recording the result.
Display state, readiness, and task kind describe progress rather than the question being asked, so they never select the template.

The templates are shared configuration rather than browser state, because the CLI and the WebUI must emit the same text.
Both templates are written together, so one configuration write never leaves a task with a stale half of the pair.
An omitted or blank template is restored to its built-in default, which keeps a configuration file written before prompts existed loading unchanged.
A template that still matches its built-in default is left out of the file.
An installation that never customized one then keeps following the built-in wording after an upgrade, rather than being pinned to whichever version first wrote the file.

A template is plain substitution over a closed vocabulary: `{{task_id}}`, `{{feature_id}}`, `{{task_title}}`, `{{task_scope}}`, and `{{task_kind}}`.
An unsupported placeholder is rejected instead of being emitted verbatim, and `{{task_id}}` is required so a rendered prompt always names its target.
Plan bodies are deliberately absent from that vocabulary: a plan may reach 1 MiB or live behind a locator, so the prompt tells the agent to read it with `prx plan TASK_ID`.
A task created without a scope renders as `(not specified)` rather than an empty line.
The templates go on to reference that scope, and a receiving agent cannot tell a blank apart from a value that failed to load.
The vocabulary is served with the stored templates so an editor presents what its own server accepts rather than a copy that could drift from it.
The built-in pair is served with them for the same reason, so an editor offering to restore the defaults shows the text it is about to write rather than an empty field.

The WebUI copies what the server renders at the moment of the copy rather than what its snapshot last recorded, so a plan registered or deleted meanwhile cannot produce the wrong prompt.
`prx prompt TASK_ID` prints the prompt body alone, with no header a caller would have to delete before using it.
The diagnostic report says whether each stored template still matches its built-in text and how long it is.
That separates an edited template from the one PRX ships without putting user-authored text into the report.
