# Architectural policy

Business rules have one application-level implementation shared by the CLI and RPC handlers.
Adapters translate that implementation instead of introducing alternate validation or status semantics.

Dependencies point inward toward domain policy.
Persistence owns storage and transactions, but it does not define competing business rules.
External providers remain replaceable behind application-facing interfaces.

The browser uses RPC and never opens SQLite or invokes the CLI.
The server remains authoritative for validation and derived business state.
The browser may translate and arrange structured state for presentation.
Known states and expected failure reasons cross RPC boundaries as enums or structured details.
Only unexpected diagnostics may remain unstructured English.
