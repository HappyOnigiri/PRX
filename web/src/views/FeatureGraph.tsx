import {
  Background,
  BackgroundVariant,
  Controls,
  MarkerType,
  ReactFlow,
  type AriaLabelConfig,
  type Connection,
  type Edge,
  type OnMove,
  type ReactFlowInstance,
} from "@xyflow/react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { mutations } from "../api";
import type { Dependency, PullRequest, Task } from "../gen/prx/v1/prx_pb";
import { useDomainMutation } from "../hooks";
import {
  maxGraphZoom,
  minGraphZoom,
  readGraphZoom,
  writeGraphZoom,
} from "../i18n/settings";
import { MutationError } from "./MutationError";
import { TaskNode, type TaskFlowNode, type TaskNodeDocument } from "./TaskNode";
import { useGraphLayout } from "./useGraphLayout";

const nodeTypes = { task: TaskNode };

interface FeatureGraphProps {
  tasks: Task[];
  dependencies: Dependency[];
  pullRequests: Map<string, PullRequest>;
  documentsByTask: Map<string, TaskNodeDocument[]>;
  onEditTask: (taskId: string) => void;
  onPreviewDocument: (document: TaskNodeDocument) => void;
  onCreateTask: () => void;
}

export function FeatureGraph({
  tasks,
  dependencies,
  pullRequests,
  documentsByTask,
  onEditTask,
  onPreviewDocument,
  onCreateTask,
}: FeatureGraphProps) {
  const { t } = useTranslation();
  const [flow, setFlow] = useState<ReactFlowInstance<TaskFlowNode>>();
  const [initialGraphZoom] = useState(readGraphZoom);
  const graphZoom = useRef(initialGraphZoom);
  const addDependency = useDomainMutation(
    ({ blocker, blocked }: { blocker: string; blocked: string }) =>
      mutations.addDependency(blocker, blocked),
  );
  const handleConnect = useCallback(
    ({ source, target }: Connection) => {
      if (addDependency.isPending || !source || !target) return;
      addDependency.mutate({ blocker: source, blocked: target });
    },
    [addDependency],
  );
  const taskTitle = useCallback(
    (taskId: string) => tasks.find((task) => task.id === taskId)?.title,
    [tasks],
  );
  const { nodes, layoutError, retryLayout } = useGraphLayout({
    tasks,
    dependencies,
    pullRequests,
    documentsByTask,
    onEditTask,
    onPreviewDocument,
  });
  const edges: Edge[] = useMemo(
    () =>
      dependencies.map((dep) => ({
        id: `${dep.blockerTaskId}-${dep.blockedTaskId}`,
        source: dep.blockerTaskId,
        target: dep.blockedTaskId,
        type: "smoothstep",
        markerEnd: { type: MarkerType.ArrowClosed },
        className: "dependency-edge",
      })),
    [dependencies],
  );
  const ariaLabelConfig = useMemo(
    () => ({
      "node.a11yDescription.default": t("workspace.flow.nodeDescription"),
      "node.a11yDescription.keyboardDisabled": t(
        "workspace.flow.keyboardDisabled",
      ),
      "edge.a11yDescription.default": t("workspace.flow.edgeDescription"),
      "controls.ariaLabel": t("workspace.flow.controls"),
      "controls.zoomIn.ariaLabel": t("workspace.flow.zoomIn"),
      "controls.zoomOut.ariaLabel": t("workspace.flow.zoomOut"),
      "controls.fitView.ariaLabel": t("workspace.flow.fitView"),
      "handle.ariaLabel": t("workspace.flow.handle"),
    }),
    [t],
  );

  useEffect(() => {
    if (nodes.length && flow) {
      const bounds = flow.getNodesBounds(nodes);
      void flow.setCenter(
        bounds.x + bounds.width / 2,
        bounds.y + bounds.height / 2,
        {
          zoom: graphZoom.current,
          duration: window.matchMedia("(prefers-reduced-motion: reduce)")
            .matches
            ? 0
            : 260,
        },
      );
    }
  }, [nodes, flow]);

  return (
    <>
      <GraphLegend
        taskCount={tasks.length}
        dependencyCount={dependencies.length}
        connectionPending={addDependency.isPending}
      />
      <GraphCanvas
        tasks={tasks}
        nodes={nodes}
        edges={edges}
        initialGraphZoom={initialGraphZoom}
        onInit={setFlow}
        onConnect={handleConnect}
        onMoveEnd={(_, viewport) => {
          graphZoom.current = viewport.zoom;
          writeGraphZoom(viewport.zoom);
        }}
        connectionPending={addDependency.isPending}
        connectionError={addDependency.error}
        taskTitle={taskTitle}
        layoutError={layoutError}
        retryLayout={retryLayout}
        onCreateTask={onCreateTask}
        ariaLabelConfig={ariaLabelConfig}
      />
    </>
  );
}

interface GraphCanvasProps {
  tasks: Task[];
  nodes: TaskFlowNode[];
  edges: Edge[];
  initialGraphZoom: number;
  onInit: (instance: ReactFlowInstance<TaskFlowNode>) => void;
  onConnect: (connection: Connection) => void;
  onMoveEnd: OnMove;
  connectionPending: boolean;
  connectionError: Error | null;
  taskTitle: (taskId: string) => string | undefined;
  layoutError: { message: string | undefined } | undefined;
  retryLayout: () => void;
  onCreateTask: () => void;
  ariaLabelConfig: Partial<AriaLabelConfig>;
}

function GraphCanvas({
  tasks,
  nodes,
  edges,
  initialGraphZoom,
  onInit,
  onConnect,
  onMoveEnd,
  connectionPending,
  connectionError,
  taskTitle,
  layoutError,
  retryLayout,
  onCreateTask,
  ariaLabelConfig,
}: GraphCanvasProps) {
  const { t } = useTranslation();
  return (
    <div
      className="graph-stage"
      data-testid="feature-graph"
      aria-busy={connectionPending}
    >
      <ReactFlow
        nodes={nodes}
        edges={edges}
        nodeTypes={nodeTypes}
        onInit={onInit}
        onConnect={onConnect}
        defaultViewport={{ x: 0, y: 0, zoom: initialGraphZoom }}
        onMoveEnd={onMoveEnd}
        minZoom={minGraphZoom}
        maxZoom={maxGraphZoom}
        nodesDraggable={false}
        nodesConnectable={!connectionPending}
        autoPanOnConnect={false}
        defaultEdgeOptions={{ animated: false }}
        ariaLabelConfig={ariaLabelConfig}
      >
        <Background
          variant={BackgroundVariant.Dots}
          gap={24}
          size={1}
          color="var(--border)"
        />
        <Controls showInteractive={false} />
      </ReactFlow>
      {connectionError && (
        <div className="graph-connection-error">
          <MutationError error={connectionError} taskTitle={taskTitle} />
        </div>
      )}
      {tasks.length === 0 && !layoutError && (
        <div className="graph-empty">
          <span>＋</span>
          <h2>{t("workspace.graphEmptyTitle")}</h2>
          <p>{t("workspace.graphEmptyDetail")}</p>
          <button onClick={onCreateTask}>{t("workspace.addTaskPlain")}</button>
        </div>
      )}
      {layoutError && (
        <div className="graph-empty" role="alert">
          <span>⚠</span>
          <h2>{t("workspace.layoutErrorTitle")}</h2>
          <p>{layoutError.message ?? t("workspace.layoutErrorFallback")}</p>
          <button onClick={retryLayout}>{t("workspace.retryLayout")}</button>
        </div>
      )}
    </div>
  );
}

function GraphLegend({
  taskCount,
  dependencyCount,
  connectionPending,
}: {
  taskCount: number;
  dependencyCount: number;
  connectionPending: boolean;
}) {
  const { t } = useTranslation();
  return (
    <div className="graph-legend">
      <p
        className={`graph-connection-help ${connectionPending ? "is-saving" : ""}`}
        aria-live="polite"
      >
        {connectionPending
          ? t("workspace.flow.connectionSaving")
          : t("workspace.flow.connectionInstruction")}
      </p>
      <div className="graph-legend-states">
        <span>
          <i className="ready" />
          {t("workspace.legend.ready")}
        </span>
        <span>
          <i className="review" />
          {t("workspace.legend.review")}
        </span>
        <span>
          <i className="conflict" />
          {t("workspace.legend.conflict")}
        </span>
        <span>
          <i className="merged" />
          {t("workspace.legend.merged")}
        </span>
      </div>
      <b>
        {t("workspace.graphSummary", {
          nodes: taskCount,
          links: dependencyCount,
        })}
      </b>
    </div>
  );
}
