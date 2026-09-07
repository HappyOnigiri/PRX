import { TaskDisplayState, type Task } from "../gen/prx/v1/prx_pb";

export interface BatchCandidate {
  task: Task;
  // Blockers that still have to be handed over together with this task. The
  // server derives which blockers are unsatisfied, so the dialog reads its
  // answer instead of walking the dependencies itself.
  pendingBlockerIds: string[];
}

// batchCandidates lists the tasks a batch may cover. A task needs a plan before
// an agent can implement it, so only designed tasks are offered at all.
//
// Without the blocked tasks, the offer is what can start right now. With them,
// it grows to the tasks whose blockers can travel in the same batch: an agent
// implements those after the work they wait for and stacks the pull requests.
// A task waiting on something the batch cannot carry — a task without a plan, or
// one in another feature — stays out, because selecting it could never become
// possible.
export function batchCandidates(
  tasks: Task[],
  includeBlocked: boolean,
): BatchCandidate[] {
  const designed = tasks.filter(
    (task) => task.displayState === TaskDisplayState.DESIGNED,
  );
  if (!includeBlocked)
    return designed
      .filter((task) => task.ready)
      .map((task) => ({ task, pendingBlockerIds: [] }));
  const offered = coverableTaskIds(designed);
  return inDependencyOrder(
    designed
      .filter((task) => offered.has(task.id))
      .map((task) => ({
        task,
        pendingBlockerIds: task.pendingBlockerTaskIds,
      })),
  );
}

// coverableTaskIds grows the ready tasks by whatever waits only on tasks
// already in the set, until nothing more can be added. A ready task has no
// unsatisfied blocker, so it enters on the first pass.
function coverableTaskIds(designed: Task[]): ReadonlySet<string> {
  const covered = new Set<string>();
  let grew = true;
  while (grew) {
    grew = false;
    for (const task of designed) {
      if (covered.has(task.id)) continue;
      if (!task.pendingBlockerTaskIds.every((id) => covered.has(id))) continue;
      covered.add(task.id);
      grew = true;
    }
  }
  return covered;
}

// inDependencyOrder puts a task behind the tasks it waits for. The copy follows
// the order the reader sees, so the batch names a blocker before the work
// stacked on it.
function inDependencyOrder(candidates: BatchCandidate[]): BatchCandidate[] {
  const placed = new Set<string>();
  const ordered: BatchCandidate[] = [];
  let remaining = candidates;
  while (remaining.length > 0) {
    const placeable = remaining.filter((candidate) =>
      candidate.pendingBlockerIds.every((id) => placed.has(id)),
    );
    // Stored dependencies are acyclic, so this is unreachable. Emitting the
    // rest in the order they arrived still offers the tasks rather than
    // looping forever.
    if (placeable.length === 0) return [...ordered, ...remaining];
    for (const candidate of placeable) {
      ordered.push(candidate);
      placed.add(candidate.task.id);
    }
    remaining = remaining.filter((candidate) => !placed.has(candidate.task.id));
  }
  return ordered;
}

// isSelectable answers whether a task can join the selection as it stands. A
// blocked task can, once everything it waits for is being handed over too.
export function isSelectable(
  candidate: BatchCandidate,
  selected: ReadonlySet<string>,
): boolean {
  return candidate.pendingBlockerIds.every((id) => selected.has(id));
}

// prunedSelection drops what the selection can no longer hand over: a task the
// dialog stopped offering, and a task whose blocker left the selection. Both
// cascade, because dropping a task can strand the work stacked on it.
export function prunedSelection(
  candidates: BatchCandidate[],
  selected: ReadonlySet<string>,
): Set<string> {
  const offered = new Set(candidates.map((candidate) => candidate.task.id));
  const kept = new Set([...selected].filter((id) => offered.has(id)));
  let shrank = true;
  while (shrank) {
    shrank = false;
    for (const candidate of candidates) {
      if (!kept.has(candidate.task.id)) continue;
      if (isSelectable(candidate, kept)) continue;
      kept.delete(candidate.task.id);
      shrank = true;
    }
  }
  return kept;
}
