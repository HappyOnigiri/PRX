// Package github isolates direct REST access to GitHub pull requests and
// provides deterministic fixture providers for tests and offline workflows. It
// is a leaf integration package that may depend on domain but not on app, store,
// cli, or rpc. Live synchronization must not persist or log tokens, and fixture
// results must remain deterministic for the same input. Read the Architecture,
// Packages, and Storage and operational boundaries sections of docs/design.md
// before changing GitHub integration.
package github
