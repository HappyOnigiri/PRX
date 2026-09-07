// document は project・feature・task のいずれか 1 つにだけ属する。各分岐で他の
// キーを never にしてあるので、呼び出し側は親を 2 つ渡せない。渡せてもサーバー
// 側で拒否される。
export type DocumentParent =
  | { projectId: string; featureId?: never; taskId?: never }
  | { projectId?: never; featureId: string; taskId?: never }
  | { projectId?: never; featureId?: never; taskId: string };
