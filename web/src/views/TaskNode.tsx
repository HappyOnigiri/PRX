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
import {
  DocumentKind,
  type TaskBlockLabel,
  type TaskDisplayState,
} from "../gen/prx/v1/prx_pb";
import { taskDisplayStateToken } from "../i18n/domain";
import type { HiddenDependencies } from "./completedTasks";
import { CopyableIdentifier } from "./CopyableIdentifier";
import { EntityIcon } from "./EntityIcon";
import { IconButton } from "./IconButton";
import { TaskPromptCopyButton } from "./TaskPromptCopyButton";
import { TaskBlockLabels, TaskStatusBadge } from "./TaskStateBadges";

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
  blockLabels: TaskBlockLabel[];
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

// React Flow は接続可否で絞らず距離だけで接続先を選ぶため、接続を拒む port が
// スナップ半径を奪ってしまう。そこで port は接続の終端を受け入れ、結果として
// 同じタスクの組に解決される。
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

// このスタブは、非表示の完了タスクへの依存を切れたエッジとして見せ、代表して
// いる非表示タスクの名前を示す。
// docs/design/webui.md を参照。
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
        <TaskStatusBadge state={data.state} />
        <div className="task-node-actions nodrag nowheel nopan">
          <CopyableIdentifier label={t("common.taskId")} value={id} valueOnly />
          {/* コピーはアーカイブ済みのタスクでも使える。エージェントに作業を
              渡すのは PRX を読むだけで、変更はしないため。 */}
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
      {/* ラベルはヘッダに並べると折り返して、先読みしたノードの高さとずれる。
          独立した行にすれば useGraphLayout の計算式と一致する。 */}
      {data.blockLabels.length > 0 && (
        <p className="task-node-blocks">
          <TaskBlockLabels labels={data.blockLabels} />
        </p>
      )}
      <h3>
        <EntityIcon kind="task" size={13} />
        {data.title}
      </h3>
      {/* 担当者はアセットではなくタイトルに属するので、責任を負う名前の
          すぐ下に置く。 */}
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

// 参照の追加は項目がもう 1 つ増えることではなく操作なので、アセット一覧の行を
// 繰り返さず、その下に独自の形で置く。
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
