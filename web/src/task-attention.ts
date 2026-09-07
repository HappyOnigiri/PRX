import { TaskBlockLabel, TaskDisplayState } from "./gen/prx/v1/prx_pb";

// 決着済みのステータス。この task で起きることはもう残っていない。
const settledStates = new Set<TaskDisplayState>([
  TaskDisplayState.COMPLETED,
  TaskDisplayState.CLOSED,
  TaskDisplayState.MERGED,
]);

// isDormantTask は、その task を今は見なくてよいかを表す。カードとグラフノードは
// この値を面の明暗で示す。決着済みのほかは未解決の依存だけが該当し、そこでは
// 何を見ても blocker が終わるまで動かせない。docs/design/webui.md を参照。
export function isDormantTask(task: {
  displayState: TaskDisplayState;
  blockLabels: TaskBlockLabel[];
}): boolean {
  if (settledStates.has(task.displayState)) return true;
  return task.blockLabels.includes(TaskBlockLabel.DEPENDENCY_UNRESOLVED);
}

// isDependencyBlockedTask は、沈んでいる理由が blocker 待ちかを表す。決着済みと
// 違って作業自体は残っているので、カードは枠を破線で残す。
export function isDependencyBlockedTask(task: {
  displayState: TaskDisplayState;
  blockLabels: TaskBlockLabel[];
}): boolean {
  if (settledStates.has(task.displayState)) return false;
  return task.blockLabels.includes(TaskBlockLabel.DEPENDENCY_UNRESOLVED);
}
