# Local trust boundary

`prx serve` is a local tool for one trusted user, not an authenticated multi-user service.
Its threat model treats the network and browser as untrusted.

The server binds to loopback by default.
Non-loopback exposure requires an explicit listen address.
Requests must be bound to the configured origin and RPC protocol so another origin cannot drive the local database.

Local file preview is limited to explicitly registered document paths.
It must not become a general filesystem reader.
Local file reads remain bounded to 1 MiB and must contain valid UTF-8 text.
URL documents are never fetched by the content-read API.

The WebUI may ask the local server to open an operating-system file chooser.
The chooser returns only a selected absolute path and does not read or register the file.
Only one chooser may be open at a time.
Cancellation is a normal result, while unavailable native helpers leave manual path entry available.
The RPC remains subject to the same Host, Origin, and Connect protocol checks as every mutation.

Production responses use restrictive browser security headers.
The server implementation owns the current header set and request-validation mechanics.
