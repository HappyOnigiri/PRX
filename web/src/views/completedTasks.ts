import {
  TaskDisplayState,
  type Dependency,
  type Task,
} from "../gen/prx/v1/prx_pb";

// グラフのフィルタはサーバーの IsTaskFinished と同じ導出状態を読む。これにより
// マージ済み・クローズ済みの PR で決着したタスクも、ブラウザ側で導出を
// 作り直すことなく完了として扱える。
function isFinishedTask(task: Task): boolean {
  return (
    task.displayState === TaskDisplayState.COMPLETED ||
    task.displayState === TaskDisplayState.CLOSED ||
    task.displayState === TaskDisplayState.MERGED
  );
}

// 表示中のタスクが依存する完了済みタスクのタイトルを、依存の向きで分けたもの。
// ノードは待っている作業と待たれている作業を示せる。
export interface HiddenDependencies {
  blockers: string[];
  blocked: string[];
}

export interface VisibleGraph {
  tasks: Task[];
  dependencies: Dependency[];
  hiddenDependencies: Map<string, HiddenDependencies>;
}

// 空のマップは共有する。グラフのレイアウトは渡されたマップの同一性を見るので、
// 描画ごとに新しく作るとキャンバスが組み直しになる。
export const emptyHiddenDependencies = new Map<string, HiddenDependencies>();

// 非表示にすると完了済みタスクはすべて消える。それを経由していた依存は捨てず、
// 両端の表示中タスクに報告する。docs/design/webui.md を参照。
export function hideFinishedTasks(
  tasks: Task[],
  dependencies: Dependency[],
): VisibleGraph {
  // 隠したタスクはタイトルも持たせる。代わりに表示するスタブが、置き換えた
  // 作業の名前を示すため。
  const hidden = new Map(
    tasks.filter(isFinishedTask).map((task) => [task.id, task.title]),
  );
  if (hidden.size === 0)
    return { tasks, dependencies, hiddenDependencies: emptyHiddenDependencies };

  const blockersOf = new Map<string, string[]>();
  const blockedOf = new Map<string, string[]>();
  for (const dependency of dependencies) {
    push(blockersOf, dependency.blockedTaskId, dependency.blockerTaskId);
    push(blockedOf, dependency.blockerTaskId, dependency.blockedTaskId);
  }

  const visibleTasks = tasks.filter((task) => !hidden.has(task.id));
  const hiddenDependencies = new Map<string, HiddenDependencies>();
  for (const task of visibleTasks) {
    const blockers = reachHidden(task.id, blockersOf, hidden);
    const blocked = reachHidden(task.id, blockedOf, hidden);
    if (blockers.length || blocked.length)
      hiddenDependencies.set(task.id, { blockers, blocked });
  }

  return {
    tasks: visibleTasks,
    dependencies: dependencies.filter(
      (dependency) =>
        !hidden.has(dependency.blockerTaskId) &&
        !hidden.has(dependency.blockedTaskId),
    ),
    hiddenDependencies,
  };
}

function push(index: Map<string, string[]>, key: string, value: string) {
  const current = index.get(key);
  if (current) current.push(value);
  else index.set(key, [value]);
}

// 探索は隠したタスクの上だけを進み、最初の表示中タスクで止まる。表示中の
// タスクに届く依存は今も描かれるので、この集計には含めない。
function reachHidden(
  start: string,
  neighbours: Map<string, string[]>,
  hidden: Map<string, string>,
): string[] {
  const found: string[] = [];
  const seen = new Set<string>();
  const queue = [...(neighbours.get(start) ?? [])];
  while (queue.length) {
    const id = queue.shift();
    const title = id === undefined ? undefined : hidden.get(id);
    if (id === undefined || title === undefined || seen.has(id)) continue;
    seen.add(id);
    found.push(title);
    queue.push(...(neighbours.get(id) ?? []));
  }
  return found;
}
