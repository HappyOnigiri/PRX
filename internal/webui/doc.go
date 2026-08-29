// Package webui serves the production Vite build embedded in the Go binary. It
// only hosts those assets and does not derive business state or invoke the CLI;
// the server and RPC layers provide the application boundary. The generated
// internal/webui/dist directory must contain only the tracked .gitkeep file in
// source control, with assets produced by the Vite build. Read the Architecture,
// Packages, and UI structure decision sections of docs/design.md before changing
// the embedded WebUI.
package webui
