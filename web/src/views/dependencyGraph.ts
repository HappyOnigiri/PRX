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
