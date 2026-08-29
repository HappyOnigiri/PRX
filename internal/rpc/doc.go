// Package rpc translates ConnectRPC requests and responses at the application
// boundary. Handlers call only the Service interface and must not own business
// validation, persistence, or synchronization rules. Fixed states, blocked
// reasons, and known errors cross the boundary as enums or structured details;
// unexpected server and GitHub errors may retain their original messages. Read
// the Architecture, Packages, and UI structure decision sections of
// docs/design.md before changing the RPC boundary.
package rpc
