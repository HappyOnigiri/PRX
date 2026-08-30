import {
  Handle,
  Position,
  useUpdateNodeInternals,
  type Node,
  type NodeProps,
} from "@xyflow/react";
import { ExternalLink, Eye, Pencil } from "lucide-react";
import { useEffect } from "react";
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

interface TaskNodePort {
  id: string;
  top: number;
}

const emptyPorts: TaskNodePort[] = [];

interface TaskNodeData extends Record<string, unknown> {
  title: string;
  assignee: string;
  state: TaskDisplayState;
  ready: boolean;
  stale: boolean;
  syncError: boolean;
  pullRequest: { label: string; url: string } | undefined;
  documents: TaskNodeDocument[];
  incomingPorts?: TaskNodePort[];
  outgoingPorts?: TaskNodePort[];
  readOnly: boolean;
  onEdit: () => void;
  onPreview: (document: TaskNodeDocument) => void;
}
export type TaskFlowNode = Node<TaskNodeData, "task">;

function TaskEdgePorts({
  incoming,
  outgoing,
}: {
  incoming: TaskNodePort[];
  outgoing: TaskNodePort[];
}) {
  return (
    <>
      {incoming.map((port) => (
        <Handle
          aria-hidden="true"
          className="task-edge-port"
          id={port.id}
          isConnectable={false}
          key={port.id}
          position={Position.Left}
          style={{ top: port.top }}
          tabIndex={-1}
          type="target"
        />
      ))}
      {outgoing.map((port) => (
        <Handle
          aria-hidden="true"
          className="task-edge-port"
          id={port.id}
          isConnectable={false}
          key={port.id}
          position={Position.Right}
          style={{ top: port.top }}
          tabIndex={-1}
          type="source"
        />
      ))}
    </>
  );
}

export function TaskNode({
  id,
  data,
  selected,
  isConnectable,
}: NodeProps<TaskFlowNode>) {
  const { t } = useTranslation();
  const updateNodeInternals = useUpdateNodeInternals();
  const incomingPorts = data.incomingPorts ?? emptyPorts;
  const outgoingPorts = data.outgoingPorts ?? emptyPorts;

  useEffect(() => {
    updateNodeInternals(id);
  }, [id, incomingPorts, outgoingPorts, updateNodeInternals]);

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
      <TaskEdgePorts incoming={incomingPorts} outgoing={outgoingPorts} />
      <div className="task-node-head">
        <div className="node-state">
          <i />
          {taskDisplayStateLabel(data.state, t)}
        </div>
        <IconButton
          icon={data.readOnly ? Eye : Pencil}
          label={
            data.readOnly
              ? t("workspace.viewTask", { title: data.title })
              : t("workspace.editTask", { title: data.title })
          }
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
