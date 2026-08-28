import { Handle, Position, type Node, type NodeProps } from "@xyflow/react";
import { useTranslation } from "react-i18next";
import { TaskDisplayState } from "../gen/prx/v1/prx_pb";
import { taskDisplayStateLabel, taskDisplayStateToken } from "../i18n/domain";

type TaskNodeData = {
  title: string;
  repository: string;
  assignee: string;
  state: TaskDisplayState;
  ready: boolean;
  stale: boolean;
};
export type TaskFlowNode = Node<TaskNodeData, "task">;

export function TaskNode({ data, selected }: NodeProps<TaskFlowNode>) {
  const { t } = useTranslation();
  return (
    <div
      className={`task-node state-${taskDisplayStateToken(data.state)} ${data.ready ? "is-ready" : ""} ${data.stale ? "is-stale" : ""} ${selected ? "is-selected" : ""}`}
    >
      <Handle type="target" position={Position.Left} />
      <div className="node-state">
        <i />
        {taskDisplayStateLabel(data.state, t)}
      </div>
      <h3>{data.title}</h3>
      <p>{data.repository || t("inspector.prNotAttached")}</p>
      <footer>
        <span>{data.assignee || t("common.unassigned")}</span>
        {data.ready && <b>{t("common.ready")}</b>}
      </footer>
      <Handle type="source" position={Position.Right} />
    </div>
  );
}
