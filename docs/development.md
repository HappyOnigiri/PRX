# Development policies

## Sources of truth

The Makefile owns the available development commands and the composition of required checks.
Tool configuration and test code own current thresholds, package scopes, fixture schemas, and scenario coverage.
Generated CLI documentation owns the current command surface.

This document records why verification exists and which properties a change must preserve.
Do not copy command inventories, test inventories, or configuration schemas here.

## Verification policy

Run `make ci` before handing off an implementation change.
The target must remain a complete local representation of required pull-request checks.

Focused checks are useful during development, but they do not replace the complete verification run.
A failing check should not prevent independent checks from reporting their results when the build tooling can continue safely.

Generated artifacts are verified against their source definitions.
Change the source and regenerate instead of repairing generated output by hand.

## Test and coverage policy

Coverage baselines prevent unintentional regression.
Raise a baseline when sustained coverage improves rather than treating spare coverage as a budget for later changes.

Core handwritten packages require every function to execute in tests.
Generated code and behavior already verified through a more appropriate public boundary may be excluded by the owning check.

Tests should verify behavior through the narrowest stable public boundary that proves the contract.
Avoid coupling policy tests to private implementation structure when a CLI, RPC, or domain boundary can express the same guarantee.

End-to-end tests cover representative cross-layer risks rather than duplicating every lower-level case.
The browser test suite owns the current scenario inventory, viewport coverage, and artifact locations.

## Deterministic external integration

Automated tests and demos use deterministic GitHub fixtures instead of live network state.
The fixture provider implementation and CLI reference own the current fixture schema and available presets.

Browser end-to-end tests start only `prx serve --demo` and use its built-in four-feature, 120-task dataset.
The large completed feature owns scale checks, while the active showcase owns state, queue, plan, and document scenarios.

Tests that exercise configuration or credential resolution must use isolated temporary configuration.
They must not depend on the real Keychain, ambient token variables, authenticated `gh` accounts, or GitHub availability.

Authentication tests use controlled HTTPS servers and explicit fake credential sources.
They verify host isolation, safe fallback, and secret-free outputs without contacting production services.

External-integration tests preserve the fail-safe synchronization and trust-boundary policies documented in `docs/design.md`.

## Version and release policy

The root `package.json` owns the PRX version used by every build surface.
Development builds append `-dev` so diagnostic output identifies their release base without claiming to be an official release.

Only the release pipeline stamps a stable version into release artifacts.
The release workflow accepts stable semantic versions and does not publish prerelease identifiers.

A release change updates the version, opens a release pull request, and derives its changelog from merged work.
Merging that pull request creates the matching tag and GitHub Release.
The workflow implementation owns credentials, branch cleanup, and other operational mechanics.
