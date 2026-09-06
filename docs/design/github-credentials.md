# GitHub credential policy

Credential methods are scoped to one normalized host and evaluated in explicit order.
An omitted method list may use documented compatibility defaults.
An explicitly empty method list disables implicit credentials.

An explicitly selected account must not inherit ambient credentials from another source.
Authorization-bearing requests must not follow redirects to another origin.

Fallback is appropriate for authentication and permission failures that another credential may resolve.
Rate limits, transport failures, and server failures do not trigger credential rotation.
The provider implementation owns the current error classification and disambiguation probes.

Caches may remember which credential method succeeded, but they must never contain credential material or token-derived secrets.
Removing a credential method invalidates its cached selection without requiring manual database repair.

Public CLI, RPC, WebUI, log, error, and cache reads remain secret-free.
Inline credentials are an explicit local-trust trade-off that favors local automation, not permission to expose stored values.
