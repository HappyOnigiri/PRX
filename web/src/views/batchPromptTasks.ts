import { TaskDisplayState, type Task } from "../gen/prx/v1/prx_pb";

export interface BatchCandidate {
  task: Task;
  // このタスクと一緒に渡す必要が残るブロッカー。未解決のブロッカーは
  // サーバーが導出するので、ダイアログは依存を自前でたどらず
  // その結果を読む。
  pendingBlockerIds: string[];
}

// batchCandidates はバッチが扱えるタスクを列挙する。設計済みのものと、要求が
// あれば単一のブロッカーを同じバッチで運べるブロック中のもの。
// docs/design/agent-prompts.md を参照。
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

// coverableTaskIds は着手可能なタスクを起点に、集合内のタスクを待つものを
// 追加できなくなるまで広げる。着手可能なタスクは未解決のブロッカーを
// 持たないため、最初の走査で集合に入る。
function coverableTaskIds(designed: Task[]): ReadonlySet<string> {
  const covered = new Set<string>();
  let grew = true;
  while (grew) {
    grew = false;
    for (const task of designed) {
      if (covered.has(task.id)) continue;
      if (task.pendingBlockerTaskIds.length > 1) continue;
      if (!task.pendingBlockerTaskIds.every((id) => covered.has(id))) continue;
      covered.add(task.id);
      grew = true;
    }
  }
  return covered;
}

// inDependencyOrder は待つ相手より後ろにタスクを置く。コピーは読み手に見える
// 順序に従うので、バッチは積まれた作業より先にブロッカーを挙げる。
function inDependencyOrder(candidates: BatchCandidate[]): BatchCandidate[] {
  const placed = new Set<string>();
  const ordered: BatchCandidate[] = [];
  let remaining = candidates;
  while (remaining.length > 0) {
    const placeable = remaining.filter((candidate) =>
      candidate.pendingBlockerIds.every((id) => placed.has(id)),
    );
    // 保存された依存は非巡回なのでここには到達しない。残りを到着順に出せば
    // 無限ループにはならず、タスクは候補として提示できる。
    if (placeable.length === 0) return [...ordered, ...remaining];
    for (const candidate of placeable) {
      ordered.push(candidate);
      placed.add(candidate.task.id);
    }
    remaining = remaining.filter((candidate) => !placed.has(candidate.task.id));
  }
  return ordered;
}

// isSelectable は現在の選択にそのタスクを加えられるかを返す。ブロック中でも
// 待っている相手をすべて一緒に渡すなら加えられる。
export function isSelectable(
  candidate: BatchCandidate,
  selected: ReadonlySet<string>,
): boolean {
  return candidate.pendingBlockerIds.every((id) => selected.has(id));
}

// prunedSelection は渡せなくなったものを選択から外す。候補でなくなったタスクと、
// ブロッカーが選択から外れたタスク。外したタスクの上に積まれた作業も土台を
// 失うため、どちらも連鎖する。
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
