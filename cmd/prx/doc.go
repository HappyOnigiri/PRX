// Package main is the PRX executable entry point and composition boundary. It
// starts the non-interactive CLI, whose setup connects the application service
// to SQLite, GitHub synchronization, ConnectRPC, and the embedded WebUI.
// The executable must keep construction at this boundary and must not duplicate
// domain validation, persistence, synchronization, or RPC translation rules.
// Read the Architecture and Packages sections of docs/design.md before changing
// the executable wiring.
package main
