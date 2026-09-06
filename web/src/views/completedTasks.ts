import {
  TaskDisplayState,
  type Dependency,
  type Task,
} from "../gen/prx/v1/prx_pb";

// The graph filter reads the derived state, the same value the server's
// IsTaskFinished reads, so a task settled by a merged or closed pull request
// counts as finished without the browser recreating that derivation.
function isFinishedTask(task: Task): boolean {
  return (
    task.displayState === TaskDisplayState.COMPLETED ||
    task.displayState === TaskDisplayState.CLOSED ||
    task.displayState === TaskDisplayState.MERGED
  );
}

// The titles of the finished tasks a visible task depends on, split by the
// direction the dependency points, so the node can say which work it waits on
// and which work waits on it.
export interface HiddenDependencies {
  blockers: string[];
  blocked: string[];
}

export interface VisibleGraph {
  tasks: Task[];
  dependencies: Dependency[];
  hiddenDependencies: Map<string, HiddenDependencies>;
}

// One shared empty map, because the graph layout keys off the identity of the
// map it is handed and a fresh one on every render would relayout the canvas.
export const emptyHiddenDependencies = new Map<string, HiddenDependencies>();

// Hiding removes every finished task, including the ones that sit between two
// unfinished tasks, so the reader never has to tell a finished node apart from
// an unfinished one. The dependency that crossed a hidden node is not lost: it
// is reported on the visible task at each end, following chains of hidden
// tasks so a whole finished stretch is counted rather than only its first node.
export function hideFinishedTasks(
  tasks: Task[],
  dependencies: Dependency[],
): VisibleGraph {
  // The hidden tasks carry their titles, because the stub that stands in for
  // one names the work it replaced.
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

// The walk only ever steps onto hidden tasks, so it stops at the first visible
// one: a dependency that reaches another visible task is still drawn and does
// not belong in the count.
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
