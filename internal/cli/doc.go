// Package cli provides the Cobra command surface with deterministic human
// output on TTY stdout and stable compact JSON on non-TTY stdout. Commands call
// the Service interface and must not duplicate application validation or
// derived-state rules. JSON success output contains only the data object;
// failures use the selected mode on stderr. Warnings and diagnostics also go to
// stderr. Read the Packages and Storage and operational boundaries sections of
// docs/design.md before changing command behavior.
package cli
