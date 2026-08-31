import {
  Handle,
  Position,
  useUpdateNodeInternals,
  type Node,
  type NodeProps,
} from "@xyflow/react";
import { ExternalLink, Eye, Pencil, Plus } from "lucide-react";
import { useEffect } from "react";
import { useTranslation } from "react-i18next";
import { DocumentKind, type TaskDisplayState } from "../gen/prx/v1/prx_pb";
import { taskDisplayStateLabel, taskDisplayStateToken } from "../i18n/domain";
import { CopyableIdentifier } from "./CopyableIdentifier";
import { IconButton } from "./IconButton";

export interface TaskNodeDocument {
  id: string;
  kind: DocumentKind;
  title: string;
  locator: string;
  isImplementationPlan: boolean;
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
  onAddReference?: (trigger: HTMLButtonElement) => void;
}
export type TaskFlowNode = Node<TaskNodeData, "task">;

// React Flow picks the connection target by distance without filtering on
// connectability, so ports that refuse connections would steal the snap radius
// from the visible handle and silently drop the connection. They accept
// connection ends instead, which resolves to the same task pair.
function TaskEdgePorts({
  incoming,
  isConnectable,
  outgoing,
}: {
  incoming: TaskNodePort[];
  isConnectable: boolean;
  outgoing: TaskNodePort[];
}) {
  return (
    <>
      {incoming.map((port) => (
        <Handle
          aria-hidden="true"
          className="task-edge-port"
          id={port.id}
          isConnectable={isConnectable}
          isConnectableEnd={isConnectable}
          isConnectableStart={false}
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
          isConnectable={isConnectable}
          isConnectableEnd={isConnectable}
          isConnectableStart={false}
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
      <TaskEdgePorts
        incoming={incomingPorts}
        isConnectable={isConnectable}
        outgoing={outgoingPorts}
      />
      <div className="task-node-head">
        <div className="node-state">
          <i />
          {taskDisplayStateLabel(data.state, t)}
        </div>
        <div className="task-node-actions nodrag nowheel nopan">
          <CopyableIdentifier label={t("common.taskId")} value={id} valueOnly />
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
            className="node-edit"
            onClick={data.onEdit}
          />
        </div>
      </div>
      <h3>{data.title}</h3>
      {data.syncError && (
        <p className="node-sync-error">{t("inspector.githubSyncError")}</p>
      )}
      {(data.pullRequest ?? (data.documents.length > 0 || !data.readOnly)) && (
        <NodeAssets data={data} />
      )}
      {data.assignee && (
        <footer>
          <span>{data.assignee}</span>
        </footer>
      )}
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

function NodeAssets({ data }: { data: TaskNodeData }) {
  const { t } = useTranslation();
  return (
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
      {[...data.documents]
        .sort(
          (left, right) =>
            Number(right.isImplementationPlan) -
            Number(left.isImplementationPlan),
        )
        .map((document) =>
          document.kind === DocumentKind.URL ? (
            <a
              className="node-asset"
              href={document.locator}
              target="_blank"
              rel="noreferrer"
              key={document.id}
            >
              <span>{document.isImplementationPlan ? "PLAN" : "URL"}</span>
              <b>{document.title || document.locator}</b>
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
              <span>
                {document.isImplementationPlan
                  ? "PLAN"
                  : document.kind === DocumentKind.LOCAL_FILE
                    ? "FILE"
                    : "MD"}
              </span>
              <b>
                {document.title ||
                  document.locator ||
                  t("inspector.referenceFallback")}
              </b>
              <Eye aria-hidden="true" focusable="false" size={14} />
            </button>
          ),
        )}
      {!data.readOnly && (
        <button
          type="button"
          className="node-asset node-asset-add"
          aria-label={t("workspace.addTaskReference", { title: data.title })}
          title={t("workspace.addTaskReference", { title: data.title })}
          onClick={(event) => {
            data.onAddReference?.(event.currentTarget);
          }}
        >
          <span>ADD</span>
          <b>{t("workspace.addReference")}</b>
          <Plus aria-hidden="true" focusable="false" size={14} />
        </button>
      )}
    </div>
  );
}
