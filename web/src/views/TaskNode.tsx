import { Handle, Position, type Node, type NodeProps } from "@xyflow/react";
import { useTranslation } from "react-i18next";
import { DocumentKind, TaskDisplayState } from "../gen/prx/v1/prx_pb";
import { taskDisplayStateLabel, taskDisplayStateToken } from "../i18n/domain";

export type TaskNodeDocument = {
  id: string;
  kind: DocumentKind;
  title: string;
  value: string;
};

type TaskNodeData = {
  title: string;
  assignee: string;
  state: TaskDisplayState;
  ready: boolean;
  stale: boolean;
  pullRequest?: { label: string; url: string };
  documents: TaskNodeDocument[];
  onEdit: () => void;
  onPreview: (document: TaskNodeDocument) => void;
};
export type TaskFlowNode = Node<TaskNodeData, "task">;

export function TaskNode({ data, selected }: NodeProps<TaskFlowNode>) {
  const { t } = useTranslation();
  return (
    <div
      className={`task-node state-${taskDisplayStateToken(data.state)} ${data.ready ? "is-ready" : ""} ${data.stale ? "is-stale" : ""} ${selected ? "is-selected" : ""}`}
    >
      <Handle type="target" position={Position.Left} />
      <div className="task-node-head">
        <div className="node-state">
          <i />
          {taskDisplayStateLabel(data.state, t)}
        </div>
        <button
          className="node-edit nodrag nopan"
          aria-label={t("workspace.editTask", { title: data.title })}
          onClick={data.onEdit}
        >
          {t("common.edit")}
        </button>
      </div>
      <h3>{data.title}</h3>
      {(data.pullRequest || data.documents.length > 0) && (
        <div className="node-assets nodrag nowheel nopan">
          {data.pullRequest && (
            <a
              className="node-asset node-asset-pr"
              href={data.pullRequest.url}
              target="_blank"
              rel="noreferrer"
            >
              <span>PR</span>
              <b>{data.pullRequest.label}</b>
              <i aria-hidden="true">↗</i>
            </a>
          )}
          {data.documents.map((document) =>
            document.kind === DocumentKind.URL ? (
              <a
                className="node-asset"
                href={document.value}
                target="_blank"
                rel="noreferrer"
                key={document.id}
              >
                <span>URL</span>
                <b>{document.title || document.value}</b>
                <i aria-hidden="true">↗</i>
              </a>
            ) : (
              <button
                className="node-asset"
                onClick={() => data.onPreview(document)}
                key={document.id}
              >
                <span>MD</span>
                <b>{document.title || document.value}</b>
                <i aria-hidden="true">⌕</i>
              </button>
            ),
          )}
        </div>
      )}
      <footer>
        <span>{data.assignee || t("common.unassigned")}</span>
        {data.ready && <b>{t("common.ready")}</b>}
      </footer>
      <Handle type="source" position={Position.Right} />
    </div>
  );
}
