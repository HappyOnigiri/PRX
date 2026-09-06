# Agent prompt policy

PRX hands one task to an AI agent that does not know PRX, so the prompt has to carry the task, its identifiers, and the commands the agent needs.
Rendering a prompt never changes task state, readiness, dependencies, or the implementation plan.
`prx prompt TASK_ID` is still an ordinary read command, so it shares the expired-interval GitHub refresh every other read runs.

Which prompt a task gets is derived from one fact only.
A task without an implementation plan gets the design prompt, which marks the task `designing` before anything else and ends by registering a plan.
A task with one gets the implementation prompt, which moves the task to `in_progress` before any change and ends by recording the result.
Both prompts set that status first so a task being designed or worked on is visible while the work is still running, rather than only once it lands.
The design prompt leaves the status alone after that first step, because registering the plan is what presents the task as designed.
Display state and readiness describe progress rather than the question being asked, so they never select the template.

A batch prompt covers several tasks at once and has its own template, because it answers a different question: which tasks are being handed over, and how the receiving agent should split them up.
It is selected by the caller asking for a batch rather than derived from any task, so it is not a third member of the pair a task chooses between.
The built-in batch template gives each task to its own SubAgent and tells that SubAgent to take its instructions from `prx prompt TASK_ID`.
The per-task text is therefore absent from the batch body: a batch stays the same length whatever it covers, and the task prompts it leads to are the same ones a reader would copy one at a time.
The caller names the feature and the tasks, and a task the feature does not own is rejected rather than rendered, so a stale selection reports what it can no longer hand over.
Rendering a batch is a read like every other prompt: it changes no task, and the WebUI is the only surface that asks for one.
It offers the tasks the server presents as designed and ready, because those are the ones whose implementation can start now; the selection itself stays with the reader.

The templates are shared configuration rather than browser state, because the CLI and the WebUI must emit the same text.
Every template is written together, so one configuration write never leaves a task with a stale half of the set.
An omitted or blank template is restored to its built-in default, which keeps a configuration file written before prompts existed loading unchanged.
A template that still matches its built-in default is left out of the file.
An installation that never customized one then keeps following the built-in wording after an upgrade, rather than being pinned to whichever version first wrote the file.

A template is plain substitution over a closed vocabulary: `{{task_id}}`, `{{feature_id}}`, `{{task_title}}`, and `{{task_scope}}`.
An unsupported placeholder is rejected instead of being emitted verbatim, and `{{task_id}}` is required so a rendered prompt always names its target.
The batch template has a vocabulary of its own, `{{task_list}}` and `{{feature_id}}`, with `{{task_list}}` required.
A task placeholder is rejected there because a batch has no single task to expand it from, and the list names every task by the identifier its SubAgent passes back to `prx prompt`.
Plan bodies are deliberately absent from that vocabulary: a plan may reach 1 MiB or live behind a locator, so the prompt tells the agent to read it with `prx plan TASK_ID`.
A task created without a scope renders as `(not specified)` rather than an empty line.
The templates go on to reference that scope, and a receiving agent cannot tell a blank apart from a value that failed to load.
The vocabulary is served with the stored templates so an editor presents what its own server accepts rather than a copy that could drift from it.
The built-in pair is served with them for the same reason, so an editor offering to restore the defaults shows the text it is about to write rather than an empty field.

The WebUI copies what the server renders at the moment of the copy rather than what its snapshot last recorded, so a plan registered or deleted meanwhile cannot produce the wrong prompt.
`prx prompt TASK_ID` prints the prompt body alone, with no header a caller would have to delete before using it.
The diagnostic report says whether each stored template, the batch one included, still matches its built-in text and how long it is.
That separates an edited template from the one PRX ships without putting user-authored text into the report.
