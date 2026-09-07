# WebUI policy

The dependency canvas was selected because causal relationships are the product's defining information.
Navigation and inspection should preserve that context instead of replacing it with a generic dashboard workflow.

Business state and credentials stay on the server.
The browser presents the values the server derived, such as a feature's read-only state, rather than recombining the flags behind them.
A feature inside an archived project is therefore read-only without the browser ever consulting the project.
Language, theme, zoom, and similar presentation-only preferences may remain browser-local.

Persistent WebUI preferences use the Settings dialog as their single change entry point.
Other screens and navigation do not duplicate controls for those preferences.
Preferences adjusted frequently during work may remain at their point of use.
Graph zoom and the sidebar's project expansion state are the current examples of this exception.

State colors are reserved for state communication rather than decoration.
Identifiers and counts may use monospace, while normal content prioritizes readability in English and Japanese.
Nonessential motion respects the reduced-motion preference.

Ordinal numbers appear only where their order is the information.
A row does not carry a running index, a record identifier, or any other number the reader cannot act on.
A field name is dropped wherever position, shape, or color already tells the reader which field a value belongs to.
The name stays in the markup for assistive technology when the visual distinction is color or placement alone.
A visible label is kept when the value would otherwise be ambiguous, or when the reader must match it against wording used elsewhere.

Controls are icon-first: a button carries an icon alone unless its meaning needs words.
Icon-only controls require an accessible name and tooltip.
Controls keep a visible label when an icon cannot communicate the target, result, or danger scope.
Pointer interactions retain a keyboard-accessible alternative.

Current screens, components, gestures, and control placement belong to the WebUI implementation and its tests.
Task search operates over the current Snapshot in the browser; its q query stays in the URL so reload, history, and sharing reproduce the view.
The status tabs of the project list and of every feature list stay in the URL for the same reason.
The navigation is a tree of projects and the features in flight inside them; which rows are collapsed is browser-local state.
Whether the task graph hides completed tasks is browser-local state as well, so the next visit reads the graph the way it was left.
Hiding removes every finished task, including one that sits between two unfinished tasks, so the reader never has to tell a finished node apart from an unfinished one.
A dependency that crossed a hidden task is not lost: it is reported on the visible task at each end.
Chains of hidden tasks are followed, so a whole finished stretch is counted rather than only its first node.
Each end presents that dependency as a severed edge naming the hidden tasks, so a task that waited on a blocker still reads as having had one.

Demo mode is injected through the served HTML metadata rather than RPC or domain state.
The WebUI keeps a non-dismissible bilingual reset warning at the top of every demo screen.
Browser-local language, theme, and zoom preferences remain outside the temporary demo environment.
