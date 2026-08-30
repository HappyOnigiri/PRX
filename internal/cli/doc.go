// Package cli provides the non-interactive Cobra command surface and stable JSON
// output envelopes. Commands call the Service interface and must not duplicate
// application validation or derived-state rules. Response-producing commands
// write schema version 2 JSON to stdout, while warnings and diagnostics go to
// stderr. Read the Packages and Storage and operational boundaries sections of
// docs/design.md before changing command behavior.
package cli
