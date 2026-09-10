import type { Edge } from "@xyflow/react";

export interface GraphPoint {
  x: number;
  y: number;
}

export interface DependencyEdgeRoute {
  points: GraphPoint[];
  sourcePortId: string;
  sourcePortTop: number;
  targetPortId: string;
  targetPortTop: number;
}

interface DependencyEdgeData extends Record<string, unknown> {
  disabled: boolean;
  label: string;
  onRemove: () => void;
  readOnly: boolean;
  removeLabel: string;
  route: DependencyEdgeRoute | undefined;
}

export type DependencyFlowEdge = Edge<DependencyEdgeData, "dependency">;

export function dependencyEdgeId(blockerTaskId: string, blockedTaskId: string) {
  return `${blockerTaskId}-${blockedTaskId}`;
}

// これから作るタスクから見た依存。direction は新しいタスクを主語に取るので、
// blocks なら新タスクが taskId をブロックし、blockedBy なら逆になる。
export interface PendingDependency {
  taskId: string;
  direction: "blocks" | "blockedBy";
}

export function dependencyPair(
  dependency: PendingDependency,
  newTaskId: string,
) {
  return dependency.direction === "blocks"
    ? { blocker: newTaskId, blocked: dependency.taskId }
    : { blocker: dependency.taskId, blocked: newTaskId };
}
