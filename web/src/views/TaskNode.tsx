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
import type { HiddenDependencies } from "./completedTasks";
import { CopyableIdentifier } from "./CopyableIdentifier";
import { EntityIcon } from "./EntityIcon";
import { IconButton } from "./IconButton";
import { StatusBadge } from "./StatusBadge";
import { TaskPromptCopyButton } from "./TaskPromptCopyButton";

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
  hasImplementationPlan: boolean;
  stale: boolean;
  syncError: boolean;
  pullRequest: { label: string; url: string } | undefined;
  documents: TaskNodeDocument[];
  hiddenDependencies?: HiddenDependencies;
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

// A hidden completed task takes its edge off the canvas with it, which would
// leave the task that waited on it looking like it never had a blocker. The
// stub keeps that connection visible as a severed edge, and names the hidden
// tasks so the reader can tell which work it stands for.
function HiddenDependencyStub({
  direction,
  titles,
}: {
  direction: "in" | "out";
  titles: string[];
}) {
  const { t } = useTranslation();
  if (titles.length === 0) return null;
  const label = t(
    direction === "in"
      ? "workspace.flow.hiddenBlockers"
      : "workspace.flow.hiddenBlocked",
    { titles: titles.join(", ") },
  );
  return (
    <span
      className={`node-hidden-dependency node-hidden-dependency-${direction}`}
      role="img"
      aria-label={label}
      title={label}
    >
      <svg viewBox="0 0 28 12" width="28" height="12" focusable="false">
        <path d="M0 6 H19" />
        <path d="M19 2 L27 6 L19 10 Z" />
      </svg>
    </span>
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
      className={`task-node state-${taskDisplayStateToken(data.state)} ${data.stale ? "is-stale" : ""} ${selected ? "is-selected" : ""}`}
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
      <HiddenDependencyStub
        direction="in"
        titles={data.hiddenDependencies?.blockers ?? []}
      />
      <HiddenDependencyStub
        direction="out"
        titles={data.hiddenDependencies?.blocked ?? []}
      />
      <div className="task-node-head">
        <StatusBadge
          className={`state-${taskDisplayStateToken(data.state)}`}
          label={taskDisplayStateLabel(data.state, t)}
        />
        <div className="task-node-actions nodrag nowheel nopan">
          <CopyableIdentifier label={t("common.taskId")} value={id} valueOnly />
          {/* Copying stays available on an archived task: handing the work to
              an agent reads PRX rather than changing it. */}
          <TaskPromptCopyButton
            taskId={id}
            hasImplementationPlan={data.hasImplementationPlan}
            size="compact"
          />
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
      <h3>
        <EntityIcon kind="task" size={13} />
        {data.title}
      </h3>
      {/* The owner belongs to the title rather than to the assets, so it reads
          directly under the name it answers for. */}
      {data.assignee && (
        <p className="node-assignee">
          <EntityIcon kind="assignee" size={13} />
          <span>{data.assignee}</span>
        </p>
      )}
      {data.syncError && (
        <p className="node-sync-error">{t("inspector.githubSyncError")}</p>
      )}
      <NodeAssets data={data} />
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

// Adding a reference is an action, not one more entry, so it sits below the
// list of assets in a shape of its own rather than repeating their row.
function NodeAssets({ data }: { data: TaskNodeData }) {
  const { t } = useTranslation();
  const hasAssets = Boolean(data.pullRequest) || data.documents.length > 0;
  return (
    <>
      {hasAssets && <NodeAssetList data={data} />}
      {!data.readOnly && (
        <button
          type="button"
          className="node-add-reference nodrag nowheel nopan"
          aria-label={t("workspace.addTaskReference", { title: data.title })}
          title={t("workspace.addTaskReference", { title: data.title })}
          onClick={(event) => {
            data.onAddReference?.(event.currentTarget);
          }}
        >
          <Plus aria-hidden="true" focusable="false" size={14} />
          <span>{t("workspace.addReference")}</span>
        </button>
      )}
    </>
  );
}

function NodeAssetList({ data }: { data: TaskNodeData }) {
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
    </div>
  );
}
