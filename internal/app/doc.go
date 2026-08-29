// Package app contains the use cases shared by the CLI and ConnectRPC layers.
// It may depend on domain and github, while persistence is accessed only through
// the Repository interface defined here; it must not depend on store, cli, or
// rpc. Validation and derived state are owned here as the single application
// boundary used by both clients. Read the Architecture, Packages, Domain
// decisions, and Storage and operational boundaries sections of docs/design.md
// before changing a use case.
package app
