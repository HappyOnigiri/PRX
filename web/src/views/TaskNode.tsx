import { Handle, Position, type Node, type NodeProps } from "@xyflow/react";

export type TaskNodeData = {
  title: string;
  repository: string;
  assignee: string;
  state: string;
  ready: boolean;
  stale: boolean;
};
export type TaskFlowNode = Node<TaskNodeData, "task">;

export function TaskNode({ data, selected }: NodeProps<TaskFlowNode>) {
  return (
    <div
      className={`task-node state-${data.state} ${data.ready ? "is-ready" : ""} ${data.stale ? "is-stale" : ""} ${selected ? "is-selected" : ""}`}
    >
      <Handle type="target" position={Position.Left} />
      <div className="node-state">
        <i />
        {data.state.replaceAll("_", " ")}
      </div>
      <h3>{data.title}</h3>
      <p>{data.repository || "PR not attached"}</p>
      <footer>
        <span>{data.assignee || "Unassigned"}</span>
        {data.ready && <b>READY</b>}
      </footer>
      <Handle type="source" position={Position.Right} />
    </div>
  );
}
