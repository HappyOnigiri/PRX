# Public contract policy

Behavior documented as a CLI, JSON, state, or dependency contract is public.
Changes to those contracts require coordinated implementation, tests, generated references, and policy updates when the policy itself changes.

Machine-readable CLI output must be deterministic and versioned, and its success data must stay free of presentation text.
Errors use stderr and leave stdout empty so automation cannot confuse a failed command with data.

| Condition | Output policy |
|---|---|
| No output flag | Concise text regardless of stdout |
| `--json` | JSON regardless of stdout |

Concise text is the default for routine inspection by people and coding agents.
Use JSON when a caller needs fields omitted from the text presentation or a stable schema for programmatic parsing.

Successful JSON commands emit their data object directly without a `schema_version`, `ok`, or `data` envelope.
Empty collections are `[]`, never `null`.
Failed JSON commands emit the versioned error object to stderr and leave stdout empty.
The current CLI response schema version is `2`.
Its error object contains `code`, `message`, and the failed command's complete help in `hint`, the only machine-readable field that carries presentation text.
Failures return a non-zero exit status.
Warnings go to stderr as text in both output modes and leave the exit status and the stdout contract untouched.
Text output does not vary with terminal width or ambient environment.
The CLI implementation and black-box tests own current field names and presentation details.

Text failures print the error followed by the same complete command help used by normal help.
An explicit `--json` makes successful help a JSON object with the complete help in `hint`.
Help succeeds without opening configuration or storage resources.

Resource commands use their shallow form for routine reads.
Feature and task commands list without an identifier and show details with one identifier.
`show` resolves a project, feature, or task public ID without the caller choosing the kind first.
Dependency and pull-request commands list when invoked without a mutation subcommand.
Document commands list without a subcommand and use `document get DOCUMENT_ID` for a detailed read.
Implementation plans use `plan TASK_ID`, agent prompts use `prompt TASK_ID`, and configuration reads use `config`, `config host`, `config auth`, or `config sync`.
Mutation operations retain explicit verbs so state-changing intent remains visible.

Mutations remain non-interactive so people and coding agents use the same surface.
A missing mutation target fails instead of reporting a successful no-op.
Destructive traversal of referenced data requires an explicit cascade request.
A cascade on a project deletes the project's own documents and every feature it holds, together with the work inside those features.
A feature cannot outlive its project, because it belongs to one.
Deleting contained work is what a cascade on a feature or a task does.
Values every invocation of an operation requires are positional operands.
Flags are reserved for optional modifiers, filters, partial updates, secret-safe input methods, execution settings, and output formats.
Flags also carry values another operand makes necessary and values chosen from mutually exclusive alternatives.
An operand value that begins with `-` is passed after `--` so it is not parsed as a flag.

Projects, features, and tasks carry public identifiers that remain distinct from their storage identifiers.
They are `P-<number>`, `F-<number>`, and `T-<number>`, and their storage UUIDs must not cross the CLI, RPC, or WebUI boundary.
An operand that accepts any of the three kinds, as `show` and `document add` do, resolves by public ID prefix, so no operand is ambiguous and a value without a known prefix is reported as not found.
A command named after one kind resolves only that kind: `project`, `feature`, and `graph` each accept the public ID of their own resource and report the operand as not found otherwise.
Documents are the deliberate exception: they have no separate public identifier,
so their storage identifier is the identifier callers pass to `document get`, `document update`, and `document delete`.
That identifier is opaque, and migrated documents may carry a value that is not formatted as a UUID.

A bulk operation may report item-level failures without discarding successful items.
Command-level failure is reserved for failure of the operation itself.
