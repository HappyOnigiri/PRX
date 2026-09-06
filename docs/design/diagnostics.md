# Diagnostic output policy

`prx debug` and its RPC are a public contract with one audience: whoever, or whatever, is asked to explain why PRX is not working.
The report leads with the problems it detected, each carrying the value that triggered it and the command whose output explains it in full.
Problem identifiers are a stable enumeration so a reader may branch on them, and the sections that follow are supporting evidence.

The report never contains credential material.
It reports environment variables by name and whether they are set, and it presents credential methods without their secrets or secret hints.
Paths are shortened to `~` under the home directory, in structured values and inside error messages alike, and a demo run reports the word `demo` instead of its temporary locations.
The diagnostic report is the one place that describes the ambient environment; every other command's text output stays independent of it.

Collecting the report changes nothing.
It never starts a synchronization run, because a refresh would clear the recorded failure the reader was asked to send, rewrite the staleness of every pull request, and block on an unreachable host.
It never creates the database file it probes for writability.

A failure in one section does not remove the others: a report is most valuable when something is broken.
`prx debug` therefore succeeds even when the database cannot be opened, and reports the failure as its storage section.

The record counts cover every stored kind, including projects and how many of them are archived, because an archived container is a reason a write was refused.

The report is bounded, and every bounded list is ordered so the same data always produces the same output, including how it is truncated.
A truncated list states how many rows were omitted rather than reading as a complete one.
Callers that need every row use `prx snapshot --json` instead.

The rendered report text crosses the RPC boundary, which is otherwise reserved for structured state.
This is a deliberate exception: the CLI and the WebUI must hand a reader exactly the same text, and a second rendering in the browser would drift from the first.
The response also carries the structured sections, so the browser presents them without re-deriving anything.
Problem descriptions are not part of that contract: the CLI writes English, and the WebUI translates the identifiers it receives.
