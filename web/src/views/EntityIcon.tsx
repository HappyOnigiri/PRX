import { CircleDot, Folder, GitBranch, User } from "lucide-react";

// One glyph per kind of record, shared by every screen. A reader who learns
// the folder on the sidebar reads the same folder in a queue row, so the icon
// replaces the field name instead of decorating it.
const entityIcons = {
  project: Folder,
  feature: GitBranch,
  task: CircleDot,
  assignee: User,
} as const;

export type EntityKind = keyof typeof entityIcons;

// The icon never carries meaning on its own: every caller keeps a name, a
// heading, or a visually hidden field label next to it for assistive
// technology, which cannot read a glyph.
export function EntityIcon({ kind, size }: { kind: EntityKind; size: number }) {
  const Icon = entityIcons[kind];
  return (
    <Icon aria-hidden="true" focusable="false" size={size} strokeWidth={1.75} />
  );
}
