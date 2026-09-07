# GitHub synchronization policy

Pull-request identity includes the normalized host.
Repositories with the same owner and name on different GitHub hosts must remain distinct.

Synchronization preserves each field's last successful value when it cannot be refreshed and marks stale data visibly.
Item-level failures leave unrelated successes intact; failure reporting depends on the last known state as described under Failure handling.

## Scheduling

GitHub synchronization is opportunistic rather than daemon-driven, so no background worker is introduced.
Commands that open the database and a visible WebUI check whether the shared interval has expired.
No refresh occurs while both the CLI and WebUI are idle.

The YAML configuration owns the shared interval and each host's GraphQL endpoint.
The interval defaults to 3600 seconds and cannot be lower than 600 seconds.
SQLite records the latest attempt and completion, and atomically grants one caller the right to run an expired refresh.

Pull requests are fetched through GraphQL in batches that group the repositories of one host into a single request.
A host whose GraphQL endpoint answers with an HTTP error falls back to fetching each pull request over REST.

## Scope

Automatic and unscoped manual refreshes include only active features, excluding completed features and those archived individually or through their project.
Explicit feature or task refreshes may still maintain archived and completed history.
An automatically completed feature does not return to active work on its own; changing its status or its tasks does that.
Attaching a pull request refreshes that pull request at once, so a task never presents freshly recorded work as stale while it waits for the next refresh.
That refresh is scoped to the task.
It is best effort and bounded by a deadline, like an automatic refresh.
Attaching succeeds even when GitHub is unreachable, and the pull request then keeps the staleness and the synchronization error that record why.
Merged and closed pull requests remain eligible so state changes and prior errors can be detected.
Only a refresh that covers every eligible pull request records a run and resets the interval; one narrowed to a feature or task leaves the recorded run status untouched.

## Failure handling

When a refresh fails after a closed or merged state is known, that state is preserved, SyncError is cleared, Stale is set, and the item is excluded from failed counts.
Open or unknown failures preserve partial fields, record SyncError, set Stale, and count as failed.

Automatic failures are best effort and never fail the command or page load that noticed the expired interval.
An automatic refresh is bounded by a deadline so an unreachable host cannot block the command that noticed the expired interval; exceeding it is recorded as an automatic failure.
They remain visible in the persisted run status and the stale state of affected pull requests.
Manual refreshes continue to return operation-level failures while preserving successful item updates.
