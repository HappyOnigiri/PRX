import {
  CircleDot,
  Folder,
  GitBranch,
  GitPullRequest,
  User,
} from "lucide-react";

// レコードの種類ごとに 1 つの図形を、全画面で共有する。サイドバーで覚えた
// フォルダはキューの行でも同じものとして読めるので、アイコンは飾りではなく
// 項目名の代わりになる。
const entityIcons = {
  project: Folder,
  feature: GitBranch,
  task: CircleDot,
  pullRequest: GitPullRequest,
  assignee: User,
} as const;

export type EntityKind = keyof typeof entityIcons;

// アイコン単体では意味を持たせない。支援技術は図形を読めないので、呼び出し側は
// 必ず名前・見出し・視覚的に隠したラベルのいずれかを隣に置く。
export function EntityIcon({ kind, size }: { kind: EntityKind; size: number }) {
  const Icon = entityIcons[kind];
  return (
    <Icon aria-hidden="true" focusable="false" size={size} strokeWidth={1.75} />
  );
}
