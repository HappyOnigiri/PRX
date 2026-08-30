import { Handle, Position, type Node, type NodeProps } from "@xyflow/react";
import { ExternalLink, Eye, Pencil } from "lucide-react";
import { useTranslation } from "react-i18next";
import { DocumentKind, type TaskDisplayState } from "../gen/prx/v1/prx_pb";
import { taskDisplayStateLabel, taskDisplayStateToken } from "../i18n/domain";
import { IconButton } from "./IconButton";

export interface TaskNodeDocument {
  id: string;
  kind: DocumentKind;
  title: string;
  value: string;
}

interface TaskNodeData extends Record<string, unknown> {
  title: string;
  assignee: string;
  state: TaskDisplayState;
  ready: boolean;
  stale: boolean;
  syncError: boolean;
  pullRequest: { label: string; url: string } | undefined;
  documents: TaskNodeDocument[];
  onEdit: () => void;
  onPreview: (document: TaskNodeDocument) => void;
}
export type TaskFlowNode = Node<TaskNodeData, "task">;

export function TaskNode({
  data,
  selected,
  isConnectable,
}: NodeProps<TaskFlowNode>) {
  const { t } = useTranslation();
  return (
    <div
      className={`task-node state-${taskDisplayStateToken(data.state)} ${data.ready ? "is-ready" : ""} ${data.stale ? "is-stale" : ""} ${selected ? "is-selected" : ""}`}
    >
      <Handle
        type="target"
        position={Position.Left}
        className="task-handle task-handle-target"
        isConnectable={isConnectable}
        tabIndex={isConnectable ? 0 : -1}
        aria-label={t("workspace.flow.blockedHandle")}
        title={t("workspace.flow.blockedHandle")}
      />
      <div className="task-node-head">
        <div className="node-state">
          <i />
          {taskDisplayStateLabel(data.state, t)}
        </div>
        <IconButton
          icon={Pencil}
          label={t("workspace.editTask", { title: data.title })}
          variant="secondary"
          size="compact"
          iconOnly
          className="node-edit nodrag nopan"
          onClick={data.onEdit}
        />
      </div>
      <h3>{data.title}</h3>
      {data.syncError && (
        <p className="node-sync-error">{t("inspector.githubSyncError")}</p>
      )}
      {(data.pullRequest ?? data.documents.length > 0) && (
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
              <ExternalLink aria-hidden="true" focusable="false" size={14} />
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
                <ExternalLink aria-hidden="true" focusable="false" size={14} />
              </a>
            ) : (
              <button
                type="button"
                className="node-asset"
                onClick={() => {
                  data.onPreview(document);
                }}
                key={document.id}
              >
                <span>MD</span>
                <b>{document.title || document.value}</b>
                <Eye aria-hidden="true" focusable="false" size={14} />
              </button>
            ),
          )}
        </div>
      )}
      <footer>
        <span>{data.assignee || t("common.unassigned")}</span>
        {data.ready && <b>{t("common.ready")}</b>}
      </footer>
      <Handle
        type="source"
        position={Position.Right}
        className="task-handle task-handle-source"
        isConnectable={isConnectable}
        tabIndex={isConnectable ? 0 : -1}
        aria-label={t("workspace.flow.blockerHandle")}
        title={t("workspace.flow.blockerHandle")}
      />
    </div>
  );
}
