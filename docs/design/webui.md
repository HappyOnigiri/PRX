# WebUI policy

The dependency canvas was selected because causal relationships are the product's defining information.
Navigation and inspection should preserve that context instead of replacing it with a generic dashboard workflow.

Business state and credentials stay on the server.
Language, theme, zoom, and similar presentation-only preferences may remain browser-local.

Persistent WebUI preferences use the Settings dialog as their single change entry point.
Other screens and navigation do not duplicate controls for those preferences.
Preferences adjusted frequently during work may remain at their point of use.
Graph zoom and the sidebar's project expansion state are the current examples of this exception.

State colors are reserved for state communication rather than decoration.
Identifiers and counts may use monospace, while normal content prioritizes readability in English and Japanese.
Nonessential motion respects the reduced-motion preference.

Controls are icon-first: a button carries an icon alone unless its meaning needs words.
Icon-only controls require an accessible name and tooltip.
Controls keep a visible label when an icon cannot communicate the target, result, or danger scope.
Pointer interactions retain a keyboard-accessible alternative.

Current screens, components, gestures, and control placement belong to the WebUI implementation and its tests.
Task search operates over the current Snapshot in the browser; its q query stays in the URL so reload, history, and sharing reproduce the view.
The status tabs of the project list and of every feature list stay in the URL for the same reason.
The navigation is a tree of projects and the features in flight inside them; which rows are collapsed is browser-local state.

Demo mode is injected through the served HTML metadata rather than RPC or domain state.
The WebUI keeps a non-dismissible bilingual reset warning at the top of every demo screen.
Browser-local language, theme, and zoom preferences remain outside the temporary demo environment.
