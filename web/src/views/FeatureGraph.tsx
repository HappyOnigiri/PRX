import {
  Background,
  BackgroundVariant,
  Controls,
  MarkerType,
  ReactFlow,
  type AriaLabelConfig,
  type Connection,
  type Edge,
  type FinalConnectionState,
  type HandleType,
  type OnMove,
  type OnReconnect,
  type ReactFlowInstance,
} from "@xyflow/react";
import { Plus, RotateCcw, TriangleAlert } from "lucide-react";
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
import { IconButton } from "./IconButton";
import { MutationError } from "./MutationError";
import { TaskNode, type TaskFlowNode, type TaskNodeDocument } from "./TaskNode";
import { useGraphLayout } from "./useGraphLayout";

const nodeTypes = { task: TaskNode };

function useDependencyConnections() {
  const addDependency = useDomainMutation(
    ({ blocker, blocked }: { blocker: string; blocked: string }) =>
      mutations.addDependency(blocker, blocked),
  );
  const removeDependency = useDomainMutation(
    ({ blocker, blocked }: { blocker: string; blocked: string }) =>
      mutations.removeDependency(blocker, blocked),
  );
  const [detaching, setDetaching] = useState(false);
  const pending = addDependency.isPending || removeDependency.isPending;
  const onConnect = useCallback(
    ({ source, target }: Connection) => {
      if (pending || !source || !target) return;
      addDependency.mutate({ blocker: source, blocked: target });
    },
    [addDependency, pending],
  );
  const onReconnect = useCallback<OnReconnect>(() => {
    setDetaching(false);
  }, []);
  const onReconnectStart = useCallback(() => {
    setDetaching(true);
  }, []);
  const onReconnectEnd = useCallback(
    (
      _event: MouseEvent | TouchEvent,
      edge: Edge,
      _handleType: HandleType,
      connectionState: FinalConnectionState,
    ) => {
      void _event;
      void _handleType;
      setDetaching(false);
      if (removeDependency.isPending || connectionState.isValid === true) {
        return;
      }
      removeDependency.mutate({ blocker: edge.source, blocked: edge.target });
    },
    [removeDependency],
  );

  return {
    adding: addDependency.isPending,
    detaching,
    error: removeDependency.error ?? addDependency.error,
    onConnect,
    onReconnect,
    onReconnectEnd,
    onReconnectStart,
    pending,
    removing: removeDependency.isPending,
  };
}

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
  const connections = useDependencyConnections();
  const taskTitle = useCallback(
    (taskId: string) => tasks.find((task) => task.id === taskId)?.title,
    [tasks],
  );
  const { nodes, layoutError, layoutPending, retryLayout } = useGraphLayout({
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
        id: dependencyEdgeId(dep.blockerTaskId, dep.blockedTaskId),
        source: dep.blockerTaskId,
        target: dep.blockedTaskId,
        type: "smoothstep",
        markerEnd: { type: MarkerType.ArrowClosed },
        className: "dependency-edge",
        reconnectable: connections.pending ? false : "source",
        ariaLabel: t("workspace.flow.dependencyEdge", {
          blocker: taskTitle(dep.blockerTaskId) ?? dep.blockerTaskId,
          blocked: taskTitle(dep.blockedTaskId) ?? dep.blockedTaskId,
        }),
      })),
    [connections.pending, dependencies, t, taskTitle],
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
        addingDependency={connections.adding}
        detachingDependency={connections.detaching}
        removingDependency={connections.removing}
      />
      <GraphCanvas
        tasks={tasks}
        nodes={nodes}
        edges={edges}
        initialGraphZoom={initialGraphZoom}
        onInit={setFlow}
        onConnect={connections.onConnect}
        onReconnect={connections.onReconnect}
        onReconnectStart={connections.onReconnectStart}
        onReconnectEnd={connections.onReconnectEnd}
        onMoveEnd={(_, viewport) => {
          graphZoom.current = viewport.zoom;
          writeGraphZoom(viewport.zoom);
        }}
        connectionPending={connections.pending}
        layoutPending={layoutPending}
        connectionError={connections.error}
        taskTitle={taskTitle}
        layoutError={layoutError}
        retryLayout={retryLayout}
        onCreateTask={onCreateTask}
        ariaLabelConfig={ariaLabelConfig}
      />
    </>
  );
}

function dependencyEdgeId(blockerTaskId: string, blockedTaskId: string) {
  return `${blockerTaskId}-${blockedTaskId}`;
}

interface GraphCanvasProps {
  tasks: Task[];
  nodes: TaskFlowNode[];
  edges: Edge[];
  initialGraphZoom: number;
  onInit: (instance: ReactFlowInstance<TaskFlowNode>) => void;
  onConnect: (connection: Connection) => void;
  onReconnect: OnReconnect;
  onReconnectStart: (
    event: React.MouseEvent,
    edge: Edge,
    handleType: HandleType,
  ) => void;
  onReconnectEnd: (
    event: MouseEvent | TouchEvent,
    edge: Edge,
    handleType: HandleType,
    connectionState: FinalConnectionState,
  ) => void;
  onMoveEnd: OnMove;
  connectionPending: boolean;
  layoutPending: boolean;
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
  onReconnect,
  onReconnectStart,
  onReconnectEnd,
  onMoveEnd,
  connectionPending,
  layoutPending,
  connectionError,
  taskTitle,
  layoutError,
  retryLayout,
  onCreateTask,
  ariaLabelConfig,
}: GraphCanvasProps) {
  const graphBusy = connectionPending || layoutPending;
  return (
    <div
      className="graph-stage"
      data-testid="feature-graph"
      aria-busy={graphBusy}
    >
      <ReactFlow
        nodes={nodes}
        edges={edges}
        nodeTypes={nodeTypes}
        onInit={onInit}
        onConnect={onConnect}
        onReconnect={onReconnect}
        onReconnectStart={onReconnectStart}
        onReconnectEnd={onReconnectEnd}
        defaultViewport={{ x: 0, y: 0, zoom: initialGraphZoom }}
        onMoveEnd={onMoveEnd}
        minZoom={minGraphZoom}
        maxZoom={maxGraphZoom}
        nodesDraggable={false}
        nodesConnectable={!graphBusy}
        autoPanOnConnect={false}
        reconnectRadius={12}
        elevateEdgesOnSelect
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
      <GraphState
        taskCount={tasks.length}
        layoutError={layoutError}
        onCreateTask={onCreateTask}
        onRetryLayout={retryLayout}
      />
    </div>
  );
}

function GraphState({
  taskCount,
  layoutError,
  onCreateTask,
  onRetryLayout,
}: {
  taskCount: number;
  layoutError: { message: string | undefined } | undefined;
  onCreateTask: () => void;
  onRetryLayout: () => void;
}) {
  const { t } = useTranslation();
  if (taskCount === 0 && !layoutError)
    return (
      <div className="graph-empty">
        <span>
          <Plus aria-hidden="true" focusable="false" size={24} />
        </span>
        <h2>{t("workspace.graphEmptyTitle")}</h2>
        <p>{t("workspace.graphEmptyDetail")}</p>
        <IconButton
          icon={Plus}
          label={t("workspace.addTaskPlain")}
          variant="primary"
          onClick={onCreateTask}
        />
      </div>
    );
  if (layoutError)
    return (
      <div className="graph-empty" role="alert">
        <span>
          <TriangleAlert aria-hidden="true" focusable="false" size={24} />
        </span>
        <h2>{t("workspace.layoutErrorTitle")}</h2>
        <p>{layoutError.message ?? t("workspace.layoutErrorFallback")}</p>
        <IconButton
          icon={RotateCcw}
          label={t("workspace.retryLayout")}
          variant="secondary"
          onClick={onRetryLayout}
        />
      </div>
    );
  return null;
}

function GraphLegend({
  taskCount,
  dependencyCount,
  addingDependency,
  detachingDependency,
  removingDependency,
}: {
  taskCount: number;
  dependencyCount: number;
  addingDependency: boolean;
  detachingDependency: boolean;
  removingDependency: boolean;
}) {
  const { t } = useTranslation();
  const instruction = removingDependency
    ? t("workspace.flow.dependencyRemoving")
    : addingDependency
      ? t("workspace.flow.connectionSaving")
      : detachingDependency
        ? t("workspace.flow.detachInstruction")
        : t("workspace.flow.connectionInstruction");
  return (
    <div className="graph-legend">
      <p
        className={`graph-connection-help ${addingDependency || removingDependency ? "is-saving" : ""}`}
        aria-live="polite"
        title={instruction}
      >
        {instruction}
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
